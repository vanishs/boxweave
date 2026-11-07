"""Independent exact-resource and no-replay tests for Agent REST mutations."""
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from bwmini.domain import Target
from bwmini.store import Store
from bwmini import rest_mutations, rest_wire


class Box:
    rest=object()
    def __init__(self, name="test-task"):
        self.target=Target("containarium","dev","default",name,
                           name+"-container","1000",name,name+"-container")
        self.state="CONTAINER_STATE_STOPPED"
        self.resources={"cpu":"2","memory":"1GB","disk":"10GB"}
    def lifecycle_inspect(self, logical):
        if self.state == "absent":
            return None
        return {"target":self.target,"state":self.state,
                "resources":self.resources}


class MutationTests(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.store=Store(Path(self.tmp.name)/"private"/"state.sqlite3")
        self.config={"mode":"coordinated","state_db":str(self.store.path),
                     "url":"http://127.0.0.1:18080","token_file":"/tmp/test"}
        self.outbound=[]
        self.row={"username":"test-task","name":"test-task-container",
                  "createdAt":"1000","state":"CONTAINER_STATE_RUNNING",
                  "resources":{"cpu":"1","memory":"1GB","disk":"10GB"}}
        self.fail=False

    def execute(self, tool="stop_container", args=None):
        parent=self
        class Client:
            def __init__(self,config):
                self.config=config
            def request(self,method,path,body=None,timeout=30):
                parent.outbound.append((method,path,body))
                if method=="GET":
                    return {"container":parent.row}
                if parent.fail:
                    raise rest_wire.WireFault("API_OUTCOME_UNKNOWN")
                return {"message":"accepted"}
        class Claims:
            def acquire(self,*a):pass
            def dispatch_intent(self,*a):pass
            def unknown(self,*a):pass
        with patch("bwmini.rest_mutations.rest_wire.Client",Client), \
             patch("bwmini.rest_mutations.coordination_bridge.published_scope",
                   return_value={"provider":"containarium","backend":"dev",
                                 "project":"default"}), \
             patch("bwmini.rest_mutations.coordination_bridge.gateway_box",
                   return_value=("a"*64,"b"*64,"test-task")), \
             patch("bwmini.rest_mutations.coordination_bridge.SharedClaims",Claims):
            return rest_mutations.dispatch(
                self.config,tool,args or {"username":"test-task"},protected=True)

    def test_stop_one_post_and_duplicate_only_status(self):
        first=self.execute()
        self.assertEqual(first["state"],"accepted")
        second=self.execute()
        self.assertEqual(second["task_id"],first["task_id"])
        self.assertEqual([m for m,_,_ in self.outbound],
                         ["GET","POST"])
        box=Box()
        with patch("bwmini.rest_mutations.coordination_bridge.SharedClaims"):
            self.assertTrue(rest_mutations.observe_one(
                self.store,box,lambda:1000+20))
        status=rest_mutations.MutationJournal(self.store).get(
            task_id=first["task_id"])
        self.assertEqual(status["state"],"succeeded")
        self.assertTrue(status["verified"])

    def test_lost_post_receipt_keeps_unknown_and_does_not_retry(self):
        self.fail=True
        first=self.execute()
        self.assertEqual(first["state"],"unknown")
        second=self.execute()
        self.assertEqual(second["task_id"],first["task_id"])
        self.assertEqual([m for m,_,_ in self.outbound],["GET","POST"])

    def test_denied_without_protected_mode_no_provider(self):
        with self.assertRaisesRegex(ValueError,"REST_COORDINATION_REQUIRED"):
            rest_mutations.dispatch(
                self.config,"stop_container",{"username":"test-task"},
                protected=False)
        self.assertEqual(self.outbound,[])

    def test_resize_disk_shrink_rejected_prewrite(self):
        with self.assertRaises(ValueError):
            self.execute("resize_container",
                         {"username":"test-task","disk":"9GB"})
        self.assertEqual([m for m,_,_ in self.outbound],["GET"])

    def test_wrong_incarnation_is_unknown_not_terminal(self):
        first=self.execute()
        box=Box()
        box.target=Target("containarium","dev","default","test-task",
                          "test-task-container","900","test-task",
                          "test-task-container")
        with patch("bwmini.rest_mutations.coordination_bridge.SharedClaims"):
            rest_mutations.observe_one(self.store,box,lambda:1020)
        self.assertEqual(rest_mutations.MutationJournal(self.store).get(
            task_id=first["task_id"])["state"],"unknown")

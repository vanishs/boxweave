"""Web and app CLI see exact original Agent REST task read-only receipts."""
import json
from pathlib import Path
import subprocess
import sys
from test_slice import Base, HASH, PASSWORD, TARGET
from bwmini.remote_create import CreateJournal
from bwmini.web import create_app


class CrossIngressStatusTests(Base):
    def test_original_create_receipt_exposed_to_authenticated_web_only(self):
        journal=CreateJournal(self.store)
        receipt,_=journal.reserve("web-reads-original",None,"a"*64)
        task=receipt["task_id"]
        app=create_app(self.s,"http://localhost:18640",self.root,secure=False)
        web=app.test_client()
        path="/api/provider-rest-tasks/"+task
        self.assertEqual(web.get(path).status_code,401)
        self.store.admin(HASH)
        login=web.post("/api/login",json={"password":PASSWORD},
                       headers={"Origin":"http://localhost:18640"})
        self.assertEqual(login.status_code,200)
        selected=web.get(path)
        self.assertEqual(selected.status_code,200)
        self.assertEqual(selected.json["task_id"],task)
        self.assertEqual(selected.json["state"],"submitted")
        self.assertEqual(web.get("/api/provider-rest-tasks/"+"f"*32).status_code,404)

    def test_cli_status_is_read_only_original_task(self):
        journal=CreateJournal(self.store)
        receipt,_=journal.reserve("cli-reads-original",None,"b"*64)
        config=self.root/"test-app.json"
        config.write_text(json.dumps({
            "mode":"fixture","target":TARGET.data(),
            "database":str(self.store.path)}))
        result=subprocess.run([
            sys.executable,"-m","bwmini","provider-rest-receipt",
            "--config",str(config),"--operation",receipt["task_id"]],
            text=True,capture_output=True,timeout=15)
        self.assertEqual(result.returncode,0,result.stderr[:200])
        self.assertEqual(json.loads(result.stdout)["task_id"],receipt["task_id"])
        self.assertEqual(journal.get(task_id=receipt["task_id"])["state"],
                         "submitted")

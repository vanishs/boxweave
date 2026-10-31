"""M0: MCP Resources are reads, not executable tools."""
import unittest
from bwmini.mcp import MCP
from bwmini.domain import Fault

class ResourceTests(unittest.TestCase):
    @staticmethod
    def client(results):
        client=object.__new__(MCP)
        events=[]
        def call(method,args):
            events.append((method,args))
            return results.pop(0)
        client.call=call
        return client, events

    def test_pagination_and_resource(self):
        c, events=self.client([
            {"resources":[{"uri":"containarium://ci-context"}],"nextCursor":"p2"},
            {"resources":[{"uri":"containarium://ci-prompt"}]},
            {"contents":[{"uri":"containarium://ci-context","text":"ok"}]}
        ])
        self.assertEqual(len(c.list_resources()),2)
        self.assertEqual(c.read_resource("containarium://ci-context"),"ok")
        self.assertEqual([a[0] for a in events],["resources/list","resources/list","resources/read"])

    def test_repeated_cursor_and_bad_resource_fail(self):
        c,_=self.client([{"resources":[],"nextCursor":"x"},{"resources":[],"nextCursor":"x"}])
        with self.assertRaises(Fault):c.list_resources()
        c,_=self.client([{"contents":[{"uri":"containarium://other","text":"private"}]}])
        with self.assertRaises(Fault):c.read_resource("containarium://ci-context")
        c,events=self.client([])
        with self.assertRaises(Fault):c.read_resource("file:///etc/passwd")
        self.assertFalse(events)

if __name__=="__main__":
    unittest.main()

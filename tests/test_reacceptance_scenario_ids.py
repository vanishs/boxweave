"""Canonical multiword system scenarios must never disappear from reports."""
from pathlib import Path
import re
import unittest

from tools.reacceptance import scenarios


class CanonicalScenarioIdentityTests(unittest.TestCase):
    def test_every_named_canonical_scenario_is_planned_or_explicitly_blocked(self):
        seed=Path(__file__).resolve().parents[1]/'docs/verification/REACCEPTANCE_SYSTEM_SCENARIOS.md'
        canonical=set(re.findall(r'^\| `(SYS-[A-Z]+(?:-[A-Z]+)*-\d{2})` P[01]',seed.read_text(),re.M))
        rows=scenarios.seed_rows(seed)
        self.assertIn('SYS-WEB-PLAN-01',canonical)
        self.assertEqual({r['id'] for r in rows},canonical)
        plan=scenarios.plan({'actions':[{'key':'stdio_mcp:list_containers'}]},seed)
        self.assertEqual({r['id'] for r in plan['scenarios']},canonical)
        web=next(r for r in plan['scenarios'] if r['id']=='SYS-WEB-PLAN-01')
        self.assertEqual(web['status'],'BLOCKED_UNCOMPILED_GRAPH')
        self.assertIsNone(web['execution_evidence'])

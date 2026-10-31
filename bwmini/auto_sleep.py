"""Constrained Platform auto-sleep toggle, not an implicit auto-resume policy."""
from .domain import Fault

def validate(args):
    if (not isinstance(args,dict) or set(args)!={"enabled","idleThresholdMinutes"}
            or type(args["enabled"]) is not bool or
            type(args["idleThresholdMinutes"]) is not int or
            not 5<=args["idleThresholdMinutes"]<=1440):
        raise Fault("AUTO_SLEEP_SPEC_INVALID",422)
    return args

def execute(service, target, args):
    validate(args)
    from .provider import text
    provider=service.provider
    if provider.mode!="mcp":raise Fault("LIVE_PROVIDER_REQUIRED")
    with provider.session(timeout=30) as m:
        tools=m.discover()
        props=tools.get("toggle_auto_sleep",{}).get("inputSchema",{}).get("properties",{})
        if not {"username","enabled","idleThresholdMinutes"}.issubset(props):
            raise Fault("AUTO_SLEEP_CONTRACT_UNSUPPORTED",409)
        result=m.tool("toggle_auto_sleep",{"username":target.logical,**args})
        output=text(result)
    if len(output.encode())>4096:raise Fault("OUTPUT_LIMIT")
    # Tool acceptance is not proof of stopped/resumed state.
    return {"result":"upstream_acknowledged","observed_lifecycle_state":"not_checked"}

package state

// successfulCreateLineageEvidence is evaluated only for an exact successful
// Create (alias c). A lost ACK has no native ID: require its atomic original
// event, separate exact recovery receipt and matching audit, never a generic
// RELEASED claim. Callers must still prove exact identity and native poststate.
const successfulCreateLineageEvidence = `(c.native_operation_id!='' OR (
 EXISTS (SELECT 1 FROM operation_events e WHERE e.operation_id=c.id AND e.state='SUCCEEDED' AND e.safe_detail='RECOVERY_CREATE_ACK_LOST_PIN_ATOMIC_NO_REPLAY')
 AND EXISTS (SELECT 1 FROM operations rr JOIN recovery_audit a ON a.operation_id=c.id AND a.at=rr.created_at AND a.reviewer=rr.actor_id AND a.decision='RECOVERY_SETTLED' AND length(a.evidence_digest)=64 AND a.evidence_digest NOT GLOB '*[^0-9a-f]*'
 JOIN operation_events re ON re.operation_id=rr.id AND re.state='SUCCEEDED' AND re.safe_detail='SETTLED:CREATE_ACK_LOST_STATIC_POSTSTATE_VERIFIED'
 WHERE json_valid(rr.safe_intent) AND json_extract(rr.safe_intent,'$.original_operation_id')=c.id AND COALESCE(json_extract(rr.safe_intent,'$.stream'),'')='' AND rr.layer=c.layer AND rr.action='operation_recover' AND rr.state='SUCCEEDED' AND rr.resource_key='recovery:'||c.id AND rr.box_uuid=c.box_uuid AND rr.incarnation=c.incarnation)
))`

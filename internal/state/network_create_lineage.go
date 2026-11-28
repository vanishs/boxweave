package state

// successfulCreateLineageEvidence is evaluated only for an exact successful
// Create (alias c). A native ACK alone is not evidence of atomic settlement.
// Scoped recovery requires the original event and its exact linked receipt
// and audit; missing native ACK cannot establish ownership.
const successfulCreateLineageEvidence = `(c.native_operation_id!=''
 AND (SELECT COUNT(*) FROM operation_events e WHERE e.operation_id=c.id AND e.state='SUCCEEDED' AND e.safe_detail IN ('ORIGINAL_CREATE_UUID_ATOMIC','RECOVERY_SCOPED_CREATE_ATOMIC_NO_REPLAY'))=1
 AND (EXISTS (SELECT 1 FROM operation_events e WHERE e.operation_id=c.id AND e.state='SUCCEEDED' AND e.safe_detail='ORIGINAL_CREATE_UUID_ATOMIC')
 OR (SELECT COUNT(*) FROM operations rr JOIN recovery_audit a ON a.operation_id=c.id AND a.at=rr.created_at AND a.reviewer=rr.actor_id
 WHERE rr.id!=c.id AND rr.layer=c.layer AND rr.action='operation_recover' AND rr.state='SUCCEEDED' AND rr.native_operation_id=''
 AND rr.resource_key='recovery:'||c.id AND rr.box_uuid=c.box_uuid AND rr.incarnation=c.incarnation
 AND rr.actor_id!='' AND rr.token_id!='' AND CAST(rr.safe_intent AS TEXT)=json_object('original_operation_id',c.id)
 AND a.decision='RECOVERY_SETTLED' AND length(a.evidence_digest)=64 AND a.evidence_digest NOT GLOB '*[^0-9a-f]*'
 AND EXISTS (SELECT 1 FROM operation_events re WHERE re.operation_id=rr.id AND re.state='SUCCEEDED' AND re.safe_detail='SETTLED:SCOPED_LIFECYCLE_COMPLETED_NO_REPLAY'))=1))`

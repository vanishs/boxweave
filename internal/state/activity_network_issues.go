package state

// activityNetworkBindingIssues describes local ledger inconsistencies, not
// native proof or permission to reconstruct a row. Link only a unique exact
// original Create; never infer attribution from a name or an IP alone.
func activityNetworkBindingIssues(q ActivityQuery, watermark int64, labelWhere string, labelArgs []any) (string, []any) {
	query := `SELECT reason,original,box_uuid,resource,state,at_ns FROM (
	 SELECT CASE
	   WHEN (SELECT COUNT(*) FROM operations z WHERE z.layer=b.project AND z.action='box_create' AND z.box_uuid=b.uuid AND z.incarnation=b.incarnation AND z.rowid<=?)>1
	     OR (o.id IS NOT NULL AND (o.resource_key!='box:'||b.name OR o.state!='SUCCEEDED'))
	     THEN 'NETWORK_CREATE_LINEAGE_MISMATCH'
	   WHEN n.box_uuid IS NULL AND EXISTS(SELECT 1 FROM network_bindings x WHERE x.project=b.project AND x.box_uuid=b.uuid)
	     THEN 'NETWORK_BINDING_IDENTITY_MISMATCH'
	   WHEN n.box_uuid IS NULL THEN 'NETWORK_BINDING_MISSING'
	   WHEN n.state!='PINNED' THEN 'NETWORK_BINDING_NOT_PINNED'
	   WHEN n.bridge='' OR n.acl='' OR n.mac='' OR n.ipv4='' OR n.ipv6=''
	     OR EXISTS(SELECT 1 FROM network_bindings x WHERE x.project=b.project AND x.box_uuid=b.uuid AND (x.incarnation!=b.incarnation OR x.nic!='eth0'))
	     THEN 'NETWORK_BINDING_IDENTITY_MISMATCH'
	   WHEN EXISTS(SELECT 1 FROM network_address_fences f WHERE f.project=b.project AND f.box_uuid=b.uuid AND
	     (f.incarnation!=b.incarnation OR f.bridge!=n.bridge OR f.state!='RESERVED' OR
         (o.id IS NOT NULL AND EXISTS(SELECT 1 FROM operation_resource_claims c WHERE c.operation_id=o.id AND c.resource_key LIKE 'create-pin:%') AND f.operation_id!=o.id) OR
	      NOT ((f.family='ipv4' AND f.address=n.ipv4) OR (f.family='ipv6' AND f.address=n.ipv6))))
	     OR EXISTS(SELECT 1 FROM network_address_fences f WHERE f.bridge=n.bridge AND
	     ((f.family='ipv4' AND f.address=n.ipv4) OR (f.family='ipv6' AND f.address=n.ipv6)) AND
	     (f.project!=b.project OR f.box_uuid!=b.uuid OR f.incarnation!=b.incarnation OR f.state!='RESERVED'))
	     THEN 'NETWORK_ADDRESS_FENCE_MISMATCH'
	   WHEN NOT EXISTS(SELECT 1 FROM network_address_fences f WHERE f.bridge=n.bridge AND f.family='ipv4' AND f.address=n.ipv4 AND f.project=b.project AND f.box_uuid=b.uuid AND f.incarnation=b.incarnation AND f.state='RESERVED')
	     OR NOT EXISTS(SELECT 1 FROM network_address_fences f WHERE f.bridge=n.bridge AND f.family='ipv6' AND f.address=n.ipv6 AND f.project=b.project AND f.box_uuid=b.uuid AND f.incarnation=b.incarnation AND f.state='RESERVED')
	     THEN 'NETWORK_ADDRESS_FENCE_MISSING'
	   ELSE '' END AS reason,
	   COALESCE(o.id,'') AS original,b.uuid AS box_uuid,
	   'network-binding:'||b.uuid||'/'||b.incarnation AS resource,
	   COALESCE(o.state,'UNVERIFIED') AS state,b.created_at AS at_ns
	 FROM owned_boxes b
	 LEFT JOIN network_bindings n ON n.project=b.project AND n.box_uuid=b.uuid AND n.incarnation=b.incarnation AND n.nic='eth0'
	 LEFT JOIN operations o ON o.id=(SELECT CASE WHEN COUNT(*)=1 THEN MIN(z.id) ELSE NULL END FROM operations z
	   WHERE z.layer=b.project AND z.action='box_create' AND z.box_uuid=b.uuid AND z.incarnation=b.incarnation AND z.rowid<=?)
	 WHERE b.project=? AND b.state='OWNED'`
	args := []any{watermark, watermark, q.Layer}
	if q.BoxUUID != "" {
		query += " AND b.uuid=?"
		args = append(args, q.BoxUUID)
	}
	query += labelWhere + ") WHERE reason!=''"
	args = append(args, labelArgs...)
	return query, args
}

// An unarchived current PINNED row without its exact OWNED incarnation is
// not a legitimate historical tombstone. Read-only diagnosis can identify
// that orphan without adopting a foreign Box or fabricating native causality.
// The exact scoped Create is linked only when unique in the signed watermark.
func activityOrphanNetworkBindings(q ActivityQuery, watermark int64, labelWhere string, labelArgs []any) (string, []any) {
	query := `SELECT 'NETWORK_ORPHAN_BINDING',COALESCE(o.id,''),n.box_uuid,
	 'network-binding:'||n.box_uuid||'/'||n.incarnation,n.state,n.observed_at
	 FROM network_bindings n LEFT JOIN operations o ON o.id=(
	   SELECT CASE WHEN COUNT(*)=1 THEN MIN(c.id) ELSE NULL END
	   FROM operations c WHERE c.layer=n.project AND c.action='box_create'
	    AND c.box_uuid=n.box_uuid AND c.incarnation=n.incarnation AND c.rowid<=?)
	 WHERE n.project=? AND n.state='PINNED'
	  AND NOT EXISTS(SELECT 1 FROM owned_boxes b WHERE b.project=n.project AND b.uuid=n.box_uuid AND b.incarnation=n.incarnation AND b.state='OWNED')`
	args := []any{watermark, q.Layer}
	if q.BoxUUID != "" {
		query += " AND n.box_uuid=?"
		args = append(args, q.BoxUUID)
	}
	query += labelWhere
	args = append(args, labelArgs...)
	return query, args
}

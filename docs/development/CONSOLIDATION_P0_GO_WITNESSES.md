# P0 Go test preservation index

Baseline: `1cfcd3da43bc47dfc42edac58d0041c02aa16e56`. All baseline tests remain retained until a reviewed replacement/obsolete-only disposition is recorded. This index lists exact Go test declarations (including tests gated by build tags), not dynamic subtests or execution results. Preserve each actual assertion; C-ID routing is conservative, not proof of equivalence.

## cmd/boxweave/doctor_test.go

P2/P3 C06/C08/C10: CLI and fixed service leaves.

`TestPR177D1CLIChild` (L15), `TestPR177D1CLINoSourceCreation` (L26)

## cmd/boxweave/h3b_test.go

P2/P3 C06/C08/C10: CLI and fixed service leaves.

`TestH3BExactApproval` (L11), `TestH3BMaintenanceProjectionNoAuthority` (L30)

## cmd/boxweave/native_peer_audit_test.go

P2/P3 C06/C08/C10: CLI and fixed service leaves.

`TestS4B2NativePeerAuditCLIRefusesMissingSQLiteWithoutCreatingIt` (L10), `TestS4B2NativePeerPurgePlanMissingDBDoesNotCreateOrMutate` (L23)

## cmd/boxweave/oauth_setup_noninteractive_test.go

P2/P3 C06/C08/C10: CLI and fixed service leaves.

`TestOAuthSetupGenerateAndPasswordFile` (L36), `TestOAuthSetupRejectsUnsafeFiles` (L77), `TestOAuthSetupRejectsInvalidOptionsAndContents` (L134), `TestOAuthSetupPersistenceFailureCleansNewFile` (L170)

## cmd/boxweave/oauth_setup_test.go

P2/P3 C06/C08/C10: CLI and fixed service leaves.

`TestOAuthPasswordSetupSurvivesExpiredCLIContext` (L17)

## cmd/boxweave/recover_apply_test.go

P2/P3 C06/C08/C10: CLI and fixed service leaves.

`TestPR177D4ReviewEnvelope` (L15)

## cmd/boxweave/recover_groups_test.go

P2/P3 C06/C08/C10: CLI and fixed service leaves.

`TestRecoveryWorkerRestrictedGroups` (L8)

## cmd/boxweave/recover_plan_test.go

P2/P3 C06/C08/C10: CLI and fixed service leaves.

`TestPR177D3PlanCLIChild` (L16), `TestPR177D3PlanCLINoSourceCreation` (L31)

## cmd/boxweave/serve_lock_test.go

P2/P3 C06/C08/C10: CLI and fixed service leaves.

`TestS2ServeLockPreventsSecondOwnerAndReopensAfterRelease` (L10), `TestS2ServeLockRefusesSymlinkOrWorldReadableFile` (L34)

## internal/control/activity_oauth_p1_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestActivityOAuthRefreshRotationRevokedGrantAndForeignLayer` (L63), `TestActivityOAuthConcurrentReadersRevokedGrantNoPostRevokeLeak` (L160), `TestActivityOAuthRefreshReplayInvalidatesSuccessor` (L209)

## internal/control/activity_orphan_native_n3_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestN3NativeMCPOrphanNetworkRestoreActivity` (L17)

## internal/control/activity_orphan_status_native_n3_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestN3NativeMCPOrphanRestoreHostStatus` (L17)

## internal/control/activity_reconnect_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestActivityRealHTTPReplyLossReopenControllerFreshActorFindsExactUnknown` (L60)

## internal/control/activity_tools_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestActivityOfficialMCPThreeViewsAndSameLayerTakeover` (L15)

## internal/control/bh019_schema_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestBH019DottedMCPFieldsAndAgentHelp` (L13)

## internal/control/box_detail_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestS4CDetailOptInWireShape` (L14)

## internal/control/continuity_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestS2SDKNativeCorrelationMetadataAndFallback` (L13), `TestS2PrivateCorrelationDoesNotAuthenticate` (L51), `TestS2RequestPrincipalRequiresExplicitSDKCredential` (L66)

## internal/control/create_ack_native_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestN3NativeMCPCreateACKLoss` (L80)

## internal/control/create_settle_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestCreatePostreadBoundedAndNeverReDispatches` (L13), `TestTerminalFailedCreateRequiresStableStoppedReadback` (L92)

## internal/control/create_shape_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestCreateNativeResourceShapeMustMatchOriginalIntent` (L9), `TestPR178CreateShapeRequiresPersistedAutostartDefault` (L35)

## internal/control/delete_ack_native_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestB5DeleteAckChild` (L67), `TestB5NativeDeleteAckLoss` (L175)

## internal/control/delete_dependencies_native_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestB5NativeMCPForwardDependentDelete` (L23)

## internal/control/delete_noack_native_n3_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestN3NativeMCPDeleteACKLostUnprovable` (L22)

## internal/control/delete_noack_resume_n3_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestN3NativeMCPDeleteACKLostRestartEvidence` (L21)

## internal/control/delete_noack_worker_e2e_n3_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestN3NativeMCPDeleteACKLostWorkerEndToEnd` (L22)

## internal/control/delete_noack_worker_native_n3_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestN3NativeMCPDeleteACKLostWorkerLiveFallback` (L22)

## internal/control/delete_noack_worker_resume_n3_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestN3NativeMCPDeleteACKLostWorkerLiveResume` (L21)

## internal/control/delete_pin_native_n3_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestN3NativeMCPModernDeleteTombstoneCut` (L22)

## internal/control/doctor_d0_extension_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestPR177D0ExtensionScopedCauseCoverage` (L10)

## internal/control/exec_archive_limits_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestExecArchiveCapacityRefusesCaptureWithoutLosingExitIdentity` (L18), `TestExecArchiveMissingDeletedBoxIsStructuredAndForeignScopeDenied` (L63)

## internal/control/exec_archive_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestExecArchiveSurvivesDeletedBoxAndExpiresOverRealMCP` (L21), `TestExecArchiveCapsOutputAndRejectsSymlink` (L126)

## internal/control/file_tools_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestS3ThreeFileToolsAndPreNativeDeniedPaths` (L13), `TestS3FileToolDescriptionsDiscloseUnsupportedConditionalModes` (L100)

## internal/control/forward_crash_native_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestB4ForwardCrashChild` (L31), `TestB4NativeOriginalCrashRecovery` (L81)

## internal/control/forward_installed_native_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestB4InstalledMCPNativeCreateListDelete` (L89)

## internal/control/h4a_navigation_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestH4AAuthenticatedSDKNavigation` (L32)

## internal/control/h4b_capacity_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestH4BCapacityNavigationDoesNotImplyFaultOrRetry` (L9)

## internal/control/ipv4_retirement_h3b_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestH3BRetirementDeferralLogLimiter` (L9)

## internal/control/ipv4_retirement_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestIPv4RetirementLoopRetriesWithoutRestart` (L9)

## internal/control/network_native_activity_native_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestN3NativeMCPFullProjectNetworkAudit` (L19)

## internal/control/network_native_activity_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestN3NetworkNativeAuditTypedScopeAndNoAdoption` (L13), `TestN3NativeNetworkObserverUnavailableNeverClaimsEmpty` (L57)

## internal/control/network_native_audit_hardening_native_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestN3NativeMCPFullInventoryHardenedReadOnly` (L15)

## internal/control/network_native_audit_unowned_native_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestN3NativeMCPUnownedProjectInstanceRedaction` (L16)

## internal/control/network_native_restart_n3_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestN3NativeMCPNetworkViewAfterProcessRestart` (L21)

## internal/control/network_native_systemd_n3_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestN3NativeInstalledSystemdMCPAcrossRestart` (L19)

## internal/control/network_recovery_native_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestS5A4NativeMCPNetworkRecovery` (L24)

## internal/control/nic_drift_native_n3_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestN3NativeMCPPhysicalNICDrift` (L22)

## internal/control/oauth_actor_s4_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestS4OAuthRefreshHTTPMCPOriginalReceiptNotRedispatched` (L21), `TestS4OAuthViewerHistoryAfterRefreshHTTPMCP` (L125)

## internal/control/oauth_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestEmbeddedOAuthMetadataDCRBrowserCallbackAndSDK` (L173), `TestEmbeddedOAuthMethodOriginAndDCRNegatives` (L252), `TestEmbeddedOAuthJSONResponsesNoSSE` (L274), `TestEmbeddedOAuthOwnedBoxChecksAndToolRefreshWithoutParams` (L302), `TestOAuthPublicHostCannotBeFakedWithForwardedHeadersAndPlainPKCE` (L337)

## internal/control/operation_diagnostic_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestS5A1OperationDiagnosticDoesNotInferOutcome` (L9)

## internal/control/operation_recover_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestS5A2HTTPRecoveryRefusesCrossLayerAndUntrustedApproval` (L13)

## internal/control/operation_tools_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestS2MCPDurableOperationReadAndActorIsolation` (L15), `TestS2InvalidCredentialCannotMutateAnyNativeTool` (L127), `TestS2MCPQueuedCancelOriginalReceiptNeverIssuesNativeID` (L163), `TestS2MCPOriginalDeleteReplaysAfterBoxGoneWithoutNativeRead` (L244), `TestS2InFlightMCPReadRechecksCredentialBeforePrivateResponse` (L311)

## internal/control/peer_recovery_native_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestS5A4NativeMCPPeerRecovery` (L19)

## internal/control/peer_tools_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestS4B3PublicPeerToolExactSchemaAndAdminBoundary` (L12)

## internal/control/pin_repair_native_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestN3NativeMCPCreatePinRepair` (L20)

## internal/control/private_read_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestS2AllPrivateReadReturnTypesRejectRevokedCredential` (L11)

## internal/control/rc8_mcp_preflight_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestRC8AgentVisibleSchemasAndSafePreflight` (L19), `TestRC8PeerScopeRefusalPreservesFailClosedClass` (L159)

## internal/control/recovery_attention_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestS5A5HTTPReconnectRecoveryNavigationAndAttention` (L15)

## internal/control/s5a1_approval_feasibility_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestS5A1ElicitationFeasibility` (L15)

## internal/control/s5a5_handoff_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestS5A5BoxHandoffLostRecoveryAndArchive` (L21)

## internal/control/server_cache_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestMCPHTTPNeverCachesPrivateOrDeniedResponses` (L10), `TestPrivateMCPWriterOverridesLateSDKCacheControl` (L41)

## internal/control/server_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestHTTPAuthorizationHostOriginAndRevocation` (L53), `TestOfficialMCPHandshakeAndS2ReadTools` (L111)

## internal/control/strict_tool_test.go

P1/P3 C03-C05/C08: MCP and original lifecycle.

`TestS2AllPublicToolSchemasRejectUnknownArguments` (L16), `TestS2NestedModelErrorsNeverBecomeNativeIntent` (L109)

## internal/diagnostic/classify_test.go

P1-P5 C01/C03/C05/C08/C11: preserve existing behavior.

`TestPR177D2EvidenceClassesAndRedaction` (L11)

## internal/diagnostic/operator_summary_test.go

P1-P5 C01/C03/C05/C08/C11: preserve existing behavior.

`TestH3BOperatorSummaryAuthorityAndHealth` (L9)

## internal/diagnostic/recover_plan_test.go

P1-P5 C01/C03/C05/C08/C11: preserve existing behavior.

`TestPR177D3ReviewDigestAndRefusals` (L26)

## internal/diagnostic/report_test.go

P1-P5 C01/C03/C05/C08/C11: preserve existing behavior.

`TestPR177D1IssueIdentityAndCoverage` (L10), `TestPR177D2InventoryJSONContract` (L31)

## internal/install/diagnostic_observe_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR177D1ArchitectureClassification` (L5), `TestPR177D1RuntimeScopeParsing` (L54)

## internal/install/diagnostic_recovery_native_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR177D4InstalledCompletedOriginal` (L16)

## internal/install/diagnostic_recovery_selector_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR177D3ScopedExplicitReadOnlySelector` (L8)

## internal/install/diagnostic_root_policy_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR177D2PrerequisiteExecutableClassification` (L9)

## internal/install/forward_policy_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestFreshInstallForwardPolicyAndManagementBind` (L8)

## internal/install/maintenance_observe_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestH1AMaintenanceMetadata` (L11), `TestH1AMaintenanceAdapter` (L55), `TestH1AMaintenanceConfigStates` (L71)

## internal/install/native_fresh_ipv4_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestFreshIPv4SelectVerifiedCapacity` (L10), `TestFreshIPv4RouteInventoryProtectsForeignAndPolicyRoutes` (L39), `TestFreshGatewayLeaseInventory` (L69)

## internal/install/native_migration_directory_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestNativeMigrationDirectoryLinks` (L5)

## internal/install/native_migration_lab_default_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery. Build constraint: `!s1lab`.

`TestProductionNativeMigrationNeverAcceptsEnvironmentFaultInjection` (L10)

## internal/install/native_package_set_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestNativePackageSetFromMetadata` (L8)

## internal/install/native_peer_acl_compare_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestNativePeerACLWholePolicyComparison` (L9)

## internal/install/native_peer_root_grant_installed_native_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B3NativeRootSeedBaselineAndExactScope` (L33), `TestS4B3NativeNonrootSeedClaimedGrantOriginal` (L99), `TestS4B3NativeInstalledRootGrantActualEffect` (L184), `TestS4B3NativeNonrootMarkCompletedGrantACKLost` (L328)

## internal/install/native_peer_root_grant_nft_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B3GrantRootKernelWitnessExactDirectionalIPv4IPv6` (L11)

## internal/install/native_peer_root_installed_e2e_native_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2NativeInstalledRootSeedReviewedGrant` (L44), `TestS4B2NativeInstalledNonrootSeedOriginal` (L127), `TestS4B2NativeInstalledAmbiguousResumeStopsAllSiblings` (L214), `TestS4B2NativeInstalledPostACLRealEffectUnknown` (L265), `TestS4B2NativeInstalledPartialUnknownProjectLock` (L331)

## internal/install/native_peer_root_integrated_native_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2NativeRootKernelPerNICDropReadback` (L14)

## internal/install/native_peer_root_integrated_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2InstalledRootKernelPerNICReadbackRefusesMissingOrAllowedRules` (L9)

## internal/install/native_peer_root_multi_installed_native_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B3NativeNonrootAdmitFollowupPeer` (L19), `TestS4B3NativeNonrootConfirmFollowupPeer` (L81)

## internal/install/native_peer_root_revoke_snapshot_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B3RevokeSnapshotCompilerPreservesUnrelatedAllowAndExactDrops` (L10)

## internal/install/native_peer_root_service_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2RootSystemdPinnedOneOperationAndNoShellInjection` (L8)

## internal/install/native_peer_root_socket_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2RootSocketPeerCredentialsAndOnlyOriginalTuple` (L39), `TestS4B2RootSocketRejectsWrongUIDAndExtraFields` (L82), `TestS4B2RootSocketOperatorPinsOneOriginalAndPlan` (L125), `TestS4B2RootSocketCrashRestartOnlyExactStaleInode` (L154), `TestS5A4RootReadOnlyApprovalScope` (L228), `TestRetirementRootReviewIsExplicitAndReadOnly` (L266)

## internal/install/native_peer_root_tombstone_b3_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B3TombstoneSurvivesNextGrantAndRegrantRemovesOnlyOwnDrop` (L8)

## internal/install/native_prepare_retirement_check_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestNativeRetirementSnapshot` (L9)

## internal/install/native_prereq_prepare_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS25NativePrepareDigestRejectsChangedNativeOrTarget` (L13), `TestS25NativePrepareProfileContainsOnlySafeDevices` (L40), `TestS25NativePrepareProjectOnlyWidensReviewedNestingAndProxy` (L57), `TestS25NativePrepareRefusesUnapprovedNonRootAccess` (L73), `TestS25NativeAutoUserProfileEligibility` (L95), `TestS25PrivateEgressBlocksRoutedForeignProjects` (L133), `TestS25PrivateEgressRefusesUnreviewedBridgeGateway` (L217)

## internal/install/native_prereq_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPrereqDevicesFailClosed` (L10), `TestPrereqACLExclusivity` (L55), `TestPrereqDigestBindsNativeIdentity` (L77), `TestConfinedProjectInvocationIgnoresOperatorIncusConfig` (L102), `TestPrereqCanonicalExcludesVolatileUsedByAndImageTimes` (L116)

## internal/install/native_s4_b1_egress_lease_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B1ExactS1ACLNativeFixture` (L34), `TestS4B1CarveOnlyOwnedBridgeForNativeExperiment` (L86), `TestS4B1RevokeStaleRulesNativeExperiment` (L146)

## internal/install/native_s4_b2_dedicated_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2NativeDedicatedBridgeACLSetup` (L16), `TestS4B2NativeDedicatedBridgeExactPeer` (L39), `TestS4B2NativeDedicatedBridgeRevoke` (L118)

## internal/install/native_s4_b2_fault_lab_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2NativeFaultLabStaleETag` (L12), `TestS4B2NativeFaultLabUnknownSameProjectConsumer` (L47)

## internal/install/native_s4_b2_policy_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2ExactPeerCompilerKeepsS1AndForeignDenials` (L41), `TestS4B2NativeExactTargetEgressAndIngress` (L116), `TestS4B2NativeRevokeBeforeIPRelease` (L180)

## internal/install/native_scriptlet_apply_native_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2ScriptletProductRootNativeApplyRollback` (L18)

## internal/install/native_scriptlet_apply_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2RootScriptletDurableApplyRollbackExactOriginal` (L73), `TestS4B2RootScriptletLostAckRecoveryAndPartialRefusal` (L114), `TestS4B2ScriptletMutationRequiresApprovedOriginalDigest` (L154)

## internal/install/native_scriptlet_atomic_journal_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2ScriptletJournalFirstIntentExclusiveAndNoStateRewind` (L22), `TestS4B2ScriptletRootLockRefusesConcurrentAndSymlink` (L73), `TestS4B2ScriptletRollbackLostAckNeverReplaysUnknown` (L111), `TestS4B2ScriptletInterruptedOriginalIntentNeverDispatches` (L133)

## internal/install/native_scriptlet_plan_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2ScriptletPreflightPlanStableAndNoLegacyCredentialLeak` (L27), `TestS4B2ScriptletPlanRefusesUnsafeInstallerTakeover` (L64), `TestS4B2ScriptletPlannerRefusesMissingNativeReceipts` (L125)

## internal/install/native_scriptlet_policy_native_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2ScriptletNativePolicySource` (L12)

## internal/install/native_scriptlet_reconcile_persistence_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2ReconcilePersistenceFailurePreservesConfirmedJournalStatus` (L28)

## internal/install/native_scriptlet_reconcile_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2ScriptletReconcileACKReadback` (L8), `TestS4B2ScriptletReconcilePreservesUnknown` (L44)

## internal/install/network_ipv4_compaction_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestIPv4CompactionRootJournalExclusiveIntent` (L12), `TestIPv4CompactionRootScopeNarrow` (L52), `TestIPv4RetirementInstallerUnitIsolation` (L77), `TestIPv4CompactionUnknownResumeReadOnly` (L101)

## internal/install/network_ipv4_retirement_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestRetirementConntrackRequiresExplicitZero` (L12), `TestRetirementKernelExactReferences` (L36)

## internal/install/network_peer_bridge_guard_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2ExclusiveNativeBridgeConsumersRefuseForeignAndUnknown` (L5)

## internal/install/network_peer_conntrack_journal_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2RootConntrackDurableOneShotReceipt` (L27), `TestS4B2RootConntrackUnknownNeverRetriesAfterNativeFailure` (L53), `TestS4B2RootConntrackCrashLeavesOriginalUnknownAndBlocksReplay` (L74), `TestS4B2RootConntrackCannotDispatchWithoutExactApproval` (L107), `TestS4B2RootConntrackFailedFinalWriteCannotReportSuccess` (L135), `TestS4B2RootConntrackConcurrentApprovalOnlyOneOriginal` (L159), `TestS4B2RootConntrackPartialVerificationPreservesUnknown` (L186), `TestS4B2RootConntrackActualSIGKILLAfterPersistedIntent` (L220), `TestS4B3GuardedRevokeCompletionFailureRetainsUnknown` (L279)

## internal/install/network_peer_conntrack_native_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2NativeRootPairConntrackAfterQuiescence` (L20)

## internal/install/network_peer_conntrack_operation_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2RootPerOriginalConntrackJournalNoReplay` (L11), `TestS4B2RootPerOriginalUncertainReceiptNeverReplay` (L45)

## internal/install/network_peer_conntrack_readback_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2ConntrackInventoryNeedsNativeZeroOrMatchingOriginal` (L10), `TestS4B2ConntrackUncertainInventoryNeverDeletes` (L50)

## internal/install/network_peer_conntrack_review_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2NativePeerConntrackReadOnlyPlanSealsExactIdentityAndHost` (L11)

## internal/install/network_peer_conntrack_store_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2ConntrackScopeMustComeFromFullSQLiteLedger` (L14)

## internal/install/network_peer_conntrack_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2RootConntrackRequiresNativeExactReverseDrops` (L33), `TestS4B2RootConntrackDeletesOnlyVerifiedPairAndNeverFlushes` (L69), `TestS4B2RootConntrackNativeZeroAndUnknownNeverRetry` (L132)

## internal/install/network_peer_foreign_raw_native_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2NativeRootRejectsForeignRawNIC` (L16)

## internal/install/network_peer_fresh_admission_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178FreshInitialGuardBeforeAnyBox` (L26), `TestPR178FreshAdmissionRefusesBeforeInspection` (L54), `TestPR178FreshAdmissionTwoReadAndPathDrift` (L80), `TestPR178FreshAdmissionDanglingSymlinkAndACL` (L119), `TestScopedLegacyRefusesEveryResidualPath` (L143)

## internal/install/network_peer_fresh_bootstrap_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178FreshBootstrapIntentBeforeEffectAndNoReplay` (L13), `TestPR178FreshJournalRefusesTamperedReceipt` (L112), `TestPR178FreshEvidenceRecompilesExactGuard` (L172)

## internal/install/network_peer_fresh_ipv6_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178FreshIPv6RouteScope` (L9)

## internal/install/network_peer_global_overlap_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2GlobalRootNetworkIPOnlyConntrackScope` (L29)

## internal/install/network_peer_grant_compiler_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B3WholeDirectionalPeerACLCompiler` (L11), `TestS4B3ACLCompilerRejectsForgedOrNoncanonicalSnapshot` (L50)

## internal/install/network_peer_guard_journal_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178GuardJournalFsyncIntentBeforeEffectAndNeverReplays` (L36), `TestPR178GuardJournalUncertainEffectAndLostACKPreserveOriginal` (L64), `TestPR178GuardJournalRejectsUnsafeOriginalScopeAndFilesystem` (L124)

## internal/install/network_peer_guard_journal_transition_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178GuardJournaledNativeDirectionNoReplay` (L11), `TestPR178GuardJournaledNativeEffectAppliedButACKLost` (L53)

## internal/install/network_peer_guard_kernel_readback_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178StrictNFTJSONKernelReadbackExactDualStackAndCounters` (L50), `TestPR178StrictNFTJSONRejectsShadowAclAndNativeDrift` (L79)

## internal/install/network_peer_guard_transition_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178NftOneDirectionReplacePreservesUnrelatedCAndD` (L21), `TestPR178NftOneDirectionRejectsMultiPeerOrDefaultRuleMutation` (L66), `TestPR178NftOneDirectionUncertainACKAndKernelReadbackNeverRetry` (L123)

## internal/install/network_peer_host_routes_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2RootHostRoutesPeerScopeRefusesUnmanagedOverlap` (L32), `TestS4B2RootHostRoutesMustUseBoundedFixedIPReader` (L85)

## internal/install/network_peer_kernel_ports_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2HostKernelBridgePortCensus` (L11)

## internal/install/network_peer_live_native_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2NativeCompiledLiveTrackedFlowDrop` (L12), `TestS4B2NativeCompiledLiveTrackedFlowRestore` (L55)

## internal/install/network_peer_revocation_native_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2NativeCompiledDropTombstoneApply` (L13), `TestS4B2NativeCompiledDropTombstoneRemove` (L59)

## internal/install/network_peer_revocation_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2RevokedPeerCompilesDualStackDropWithoutGrant` (L30), `TestS4B2RevokedAndActivePeerDisjointAndDeterministic` (L67), `TestS4B2EmptyTombstonePreservesExactOldPolicy` (L117), `TestS4B2PeerACLCompilerRejectsGatewayOrIdentityDrift` (L131)

## internal/install/network_peer_root_audit_snapshot_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2RootAuditSnapshotDetectsSameSizeSecurityDrift` (L10)

## internal/install/network_peer_root_audit_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2NativeRootBridgeAudit` (L16), `TestS4B2RootBridgeAuditRejectsUntrustedScope` (L78)

## internal/install/network_peer_scoped_acl_cas_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178ScopedNativeACLETagCASGrantRevokeNeverTouchesCD` (L85), `TestPR178ScopedNativeACLETagCASRefusesUntrustedPreimage` (L137), `TestPR178ScopedNativeACLETagCASACKLossAndFaultsNeverRetry` (L204)

## internal/install/network_peer_scoped_acl_preimage_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178NativeRootACLFullProjectReadbackExactBeforeImage` (L38), `TestPR178NativeRootACLPreimageRejectsForeignAcceptAndDrift` (L65), `TestPR178NativeRootACLRegrantRequiresExistingExactDrop` (L170)

## internal/install/network_peer_scoped_boot_policy_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178N3BootPolicyJournalChain` (L13)

## internal/install/network_peer_scoped_direction_native_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178N3NativeDirectionGuard` (L147)

## internal/install/network_peer_scoped_effect_journal_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178N3WholeOriginalJournal` (L12)

## internal/install/network_peer_scoped_effect_sequence_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178FreshScopedPeerGuardedEffectSequence` (L8), `TestPR178FreshScopedEffectSequenceRejectsUnprovenOwnerOrCDDelta` (L60), `TestPR178FreshSequenceKeepsAllBootAutostartPreferences` (L105)

## internal/install/network_peer_scoped_execute_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178N3ComposedScopedEffects` (L14)

## internal/install/network_peer_scoped_guard_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178GuardNonDisruptiveDirectedDualStackAtomicText` (L29), `TestPR178GuardRefusesUnsafeNativeInventoryAndSnapshots` (L77)

## internal/install/network_peer_scoped_inventory_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178NativeScopedGuardCompleteProjectInventory` (L63), `TestPR178NativeScopedGuardRejectsIncompleteAndForeignInventory` (L81)

## internal/install/network_peer_scoped_kernel_ports_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178ExactKernelPortsOnlyOwnedRunningBoxVeths` (L33), `TestPR178ExactKernelPortsRejectsForeignRawAndDrift` (L53)

## internal/install/network_peer_scoped_partial_native_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178N4InstalledPartialACL` (L34)

## internal/install/network_peer_scoped_policy_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178ScopedACLUnaffectedOwnerHasIdenticalPolicy` (L40), `TestPR178ScopedACLRejectsAllUntrustedSnapshotsAndOwnerDrift` (L111), `TestPR178ScopedACLNamePinnedToProjectBridgeAndNativeUUID` (L141), `TestPR178N3ReversePolicyPreservesIndependentGrant` (L171)

## internal/install/network_peer_scoped_root_kernel_collector_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178RootSDKSysfsNFTDoublePreimageCompleteNoMutation` (L44), `TestPR178RootSDKSysfsNFTRefusesPartialAndTransientNativeEvidence` (L84), `TestPR178RootSDKSysfsNFTRefusesChangingKernelAndSDKState` (L137), `TestPR178RootSDKSysfsNFTRejectsMissingWitnessOrRecoveryOriginal` (L205)

## internal/install/network_peer_scoped_root_sdk_collector_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178RootSDKReadOnlyCompleteTenantAndRootACLNoNativeMutation` (L120), `TestPR178RootSDKReadOnlyRejectsUntrustedAndDriftingNativeScope` (L150), `TestPR178RootSDKReadOnlyRefusesCancelAndMissingProtectedStore` (L211)

## internal/install/network_peer_scoped_root_transition_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178NativeRootScopeCombinesV2SQLiteAndIncusFourOwners` (L104), `TestPR178NativeRootScopeRevokeAndRegrantKeepExactTombstones` (L130), `TestPR178NativeRootScopedPlanFailsClosedOnForeignOrDrift` (L155)

## internal/install/network_peer_scoped_service_dependency_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestPR178N4DegradedControllerDependency` (L8)

## internal/install/network_peer_state_audit_native_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2NativeStoreLedgerBridgeAudit` (L18)

## internal/install/network_peer_state_audit_race_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestS4B2RootAuditRefusesSQLiteAndNativeInterleaving` (L57), `TestS4B2RootAuditPreventsUnpinnedAndCancelledNativeRead` (L105)

## internal/install/scoped_retirement_lxc_test.go

P1/P2/P3 C04-C08/C10: native policy, install, retirement, recovery.

`TestScopedRetirementLXCPolicyExact` (L5)

## internal/native/bh018_127_native_outcome_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestBH018NativeSpecialExitIsFinishedButNotGuestSuccess` (L16), `TestBH018OtherNativeFailureRemainsUnknown` (L52), `TestBH018NativeExecOutputBoundToOriginalBoxAndOperation` (L85), `TestBH018SuccessfulNativeExecMayHaveNonzeroGuestExit` (L118), `TestBH018NativeTypedIntExitOutsideRecordedRangeRejected` (L134)

## internal/native/box_detail_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestS4CExactFullInstanceObservation` (L26), `TestS4CFullReadFailsClosedWithoutExtensionAndOnNativeError` (L62)

## internal/native/box_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestS2NetworkMustBeVerifiedBeforeFirstStart` (L12), `TestS25DefaultProfileMustNotOpenEveryRawNativeGuest` (L59), `TestS2ExecOutputIsNotGuestExitZeroByDefault` (L86), `TestS2RecordedOutputDoHTTPUsesExactSDKURLAndRestrictedProject` (L114), `TestRC3IncusLocalAndCatalogImagesNotUbuntuLocked` (L135), `TestS2GuestTimeoutArgvHasOptionsSeparatorBeforeDuration` (L146), `TestS2OwnedNICInternetToggleNeverEditsSharedACL` (L168), `TestS2EffectiveNICAndDiskAlwaysReadRegardlessOfMapIteration` (L214), `TestS2MalformedNativeExecReturnAndStreamMetadataFailClosed` (L241), `TestS2GuestReadyJSONTriStateDoesNotInferFromNativeRunning` (L288), `TestS2NativeGuestReadinessOnlyGuestDeclaredAndCurrent` (L316)

## internal/native/create_operation_inactive_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestOriginalCreateOperationListingCannotHideActiveOriginal` (L12)

## internal/native/exec_quota_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestRC8ExecSpoolQuotaHeadroom` (L10), `TestRC8ExactIncusRecordedExecOpenQuotaFailure` (L41)

## internal/native/file_native_integration_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestS3RealIncusSFTPFileNegatives` (L18)

## internal/native/file_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestS3CanonicalGuestFilePathDenyMagicAndTraversal` (L11), `TestS3NativeFileTypeClassificationDoesNotFollowLinks` (L34), `TestS3SFTPReadBudgetCannotDecodeUnboundedDirectory` (L56)

## internal/native/forward_native_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestB4NativeOwnedProxyTCPUDP` (L58)

## internal/native/forward_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestB4ForwardSpecAndExactProxyDevice` (L8), `TestB4HostLocalForwardPolicy` (L69), `TestForwardOperatorAllPolicy` (L92)

## internal/native/h4b_capacity_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestH4BAddressCapacityUnionAndRetention` (L10)

## internal/native/instance_options_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestBH019DottedOptionAdmissionFailsClosed` (L14), `TestBH019PresenceAndNoRawPassthrough` (L38), `TestBH019BPFDependenciesAndProjectGate` (L64), `TestBH019AllowlistedLowlevelOnly` (L78), `TestPR178CreateAutostartDefaultPreservesExplicitChoice` (L110)

## internal/native/native_journal_integration_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestS1IncusNativeJournalRecovery` (L113)

## internal/native/network_project_scan_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestN3NativeProjectInventoryScopeAndIncompleteCases` (L41), `TestN3NativeProject1024ObjectsAnd1025FailureNoPartial` (L94)

## internal/native/network_scan_race_n3_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestN3NativeSnapshotFullNetworkLeaseRaceIsFailClosed` (L98), `TestN3NativeCancelledOrExpiredInventoryNeverClears` (L126)

## internal/native/peer_acl_isolation_native_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestS4B2NativeSDKSharedACLQuiescence` (L17)

## internal/native/peer_acl_isolation_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestS4B2QuiescentSharedACLStopsEveryNICBeforePUT` (L86), `TestS4B2QuiescentSharedACLRejectsUnsafeBindingsBeforeSideEffect` (L106), `TestS4B2QuiescentSharedACLRootCheckMandatory` (L146), `TestS4B2QuiescentSharedACLLostACKNeverResumes` (L155)

## internal/native/peer_static_binding_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestStaticPeerBindingSnapshot` (L11), `TestPeerStatePrefixesRedacted` (L46), `TestCreateAddressCapacityExhaustionIsTypedAndPreDispatch` (L72)

## internal/native/profile_config_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestS25OfficialIncusMultiUserProfileConfigOnly` (L5)

## internal/native/project_capabilities_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestProjectCapabilityReportAndRefusals` (L10), `TestForwardPolicyModeAndDenial` (L49)

## internal/native/rc3_image_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestRC3IncusVersionBaseline` (L5), `TestRC3GuestImageSelectionNoArbitraryURLs` (L19)

## internal/native/readiness_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestS4CReadinessNativeAddressAndExitProof` (L9)

## internal/native/storage_driver_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestRC2NativeStorageAdmissionIgnoresDriver` (L10)

## internal/native/storage_filesystem_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestS1CustomDirPoolOnHostRootCannotProveBacking` (L11), `TestS1DefaultRootDirPoolSpaceIsReadFromSelectedFilesystem` (L31), `TestS1DirPoolMissingSourceAndSymlinkAreRejected` (L44), `TestS1PoolNameMayNotEscapeCanonicalPath` (L61), `TestS2QuotaMountSourceMustBeRealEnabledExt4OrXFS` (L71), `TestS25DirQuotaEvidenceIsNotGeneralFilesystemSupport` (L98)

## internal/native/storage_quota_witness_test.go

P1/P2 C03-C06/C09: actual native constraints.

`TestRC8HostRootQuotaWitnessFailClosed` (L8), `TestNearFullHostQuotaWitnessRequiresWriteRefusal` (L67)

## internal/networklimits/peer_history_test.go

P1-P5 C01/C03/C05/C08/C11: preserve existing behavior.

`TestScopedPeerHistoryReserveRevocation` (L5), `TestH4BGrantBudgetMatchesAdmission` (L36)

## internal/peerscope/name_test.go

P1-P5 C01/C03/C05/C08/C11: preserve existing behavior.

`TestPR178ScopedACLNameStableAndUnique` (L5)

## internal/recovery/create_test.go

P1-P5 C01/C03/C05/C08/C11: preserve existing behavior.

`TestN3StagedCreateRecoveryProofMatrix` (L41), `TestN3CompletedCreateNativePinDiagnosis` (L134)

## internal/recovery/doctor_d0_extension_test.go

P1-P5 C01/C03/C05/C08/C11: preserve existing behavior.

`TestPR177D0ExtensionScopedWitnessDispatchBoundary` (L13)

## internal/recovery/lifecycle_pin_test.go

P1-P5 C01/C03/C05/C08/C11: preserve existing behavior.

`TestN3LifecycleNativePinMatrix` (L24)

## internal/recovery/plan_observation_test.go

P1-P5 C01/C03/C05/C08/C11: preserve existing behavior.

`TestPR177D3PlanObservationUsesReadOnlyPredicates` (L34)

## internal/recovery/reviewed_apply_test.go

P1-P5 C01/C03/C05/C08/C11: preserve existing behavior.

`TestPR177D4ServiceReobservesAndRechecks` (L13)

## internal/recovery/service_test.go

P1-P5 C01/C03/C05/C08/C11: preserve existing behavior.

`TestS5A2DomainProofMatrix` (L51), `TestS5A4GenericReleaseAuditDoesNotOverrideNetworkBarrier` (L140)

## internal/state/acl_sync_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B2SynchronousACLTypedSuccessNoInventedNativeID` (L23), `TestS4B2SynchronousACLUnknownPreservesLockAndNoRetry` (L44), `TestS4B2SynchronousACLFailedPostreadNeverClaimsSuccess` (L74), `TestS4B2SynchronousACLNoIsolationRefusesSubmission` (L94), `TestS4B2SynchronousACLFailedBarrierQuarantinesWithoutPUT` (L114), `TestS4B2SynchronousACLIsolationRunsBeforePUT` (L138)

## internal/state/activity_issues_clarity_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestActivityIssuesSeparatesLiveReviewReleasedHistoryAndCoverage` (L9)

## internal/state/activity_network_issues_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestN3ActivityNetworkBindingIssues` (L9)

## internal/state/activity_network_orphans_n3_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestN3ActivityModernCreateLineageAndScopedOrphan` (L27), `TestN3ActivityNativeBridgeUniquenessRejectsCrossProjectCollision` (L68)

## internal/state/activity_p1_stress_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestActivityP1History100kOptIn` (L13)

## internal/state/activity_query_pagination_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestActivityIssuesCoverPeerForwardFenceAndRealBoxUUID` (L10), `TestActivityOverviewCursorAdmissionWatermarkAndTamper` (L70), `TestActivityIssuesSameTimestampKeysetAndScope` (L107)

## internal/state/activity_query_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestActivityOverviewIncludesPeerTargetClaimWithoutForeignLayer` (L35), `TestActivityIssuesFindOldUnknownAndRootPeerDespiteLegacyRelease` (L60), `TestActivityEvidenceReadsOrderedSanitizedEventsAndClaims` (L106), `TestActivityPeerEvidenceVerifiedLinkNotNativePermission` (L139), `TestActivityQueryRejectsInvalidAndTruncates` (L167), `TestActivityRevokedTokenNeverReceivesReadback` (L192)

## internal/state/admission_navigation_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestH4AExactAdmissionBlockerPriorityAndScope` (L10)

## internal/state/create_predispatch_pin_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestN3WorkerVerifiedNoDispatchCreateReturnsExactReservation` (L33), `TestN3CreateUncertainOrDispatchedFailureRetainsStaticHolds` (L75), `TestN3NoEffectCreateReservationReleaseAtomicFaultAndConflict` (L99)

## internal/state/create_recovery_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestUnknownCreateReviewedNativeIdentityBindAndNoReplay` (L11), `TestUnknownCreateMalformedIntentFailsClosed` (L95), `TestUnknownCreatePostCrashNoNativeAckStillReviewable` (L125), `TestUnknownCreateBindingAuditFailureIsAtomic` (L188)

## internal/state/delete_dependencies_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestB5DeleteForwardParentChildAtomicAndReplay` (L54), `TestB5DeletePeerRootLinkedProofAndRetainedDropHistory` (L157), `TestB5DeleteRestartPartialCleanupUnknownFences` (L246), `TestB5ReviewKnownTerminalDeleteAtomicPinnedTombstone` (L293)

## internal/state/diagnostic_ledger_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR177D2ScopedRetiredAndLegacyRoutes` (L13), `TestPR177D2ScopedUnknownAndFailedRecoveryHistory` (L35), `TestPR177D2CapacityAndRedactedOAuth` (L77), `TestPR177D2ReleasedHistoryAndGuestExitSeparate` (L100)

## internal/state/diagnostic_recovery_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR177D3ScopedSnapshotProofAndRevision` (L11)

## internal/state/diagnostic_snapshot_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR177D1WriterProcess` (L18), `TestPR177D1LiveColdAndCrashSnapshots` (L138), `TestPR177D1SnapshotRefusals` (L236), `TestPR177D1SnapshotSchemaAndEmptyScoped` (L320)

## internal/state/doctor_d0_experiment_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR177D0ReadOnlyReaderFootprint` (L30), `TestPR177D0ScopedLedgerDiagnosticBoundary` (L112)

## internal/state/doctor_d0_extension_boundaries_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR177D0ExtensionLiveReadTransactionEpoch` (L14), `TestPR177D0ExtensionScopedRetirementStatusMismatch` (L57), `TestPR177D0ExtensionOriginalCredentialRevocation` (L85)

## internal/state/doctor_d0_extension_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR177D0ExtensionPrivateWALSnapshot` (L17), `TestPR177D0ExtensionScopedRecoveryReceiptAndReview` (L99)

## internal/state/doctor_d0_immutable_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR177D0ImmutableIsNotLiveWALReader` (L14)

## internal/state/exec_gc_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS2ExecGCRequiresSevenDaysAndExactTerminalOwnership` (L52), `TestS2ExecGCInterruptedNativeDELETEBecomesReviewOnlyNoRetry` (L117), `TestS2ExecGCRejectsTamperedNativePaths` (L164), `TestS2ExecGCManualChallengeRecoveryNeverRedispatchesOriginal` (L182)

## internal/state/file_create_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS3FileCreateAtomicTerminalAndOriginalIdempotency` (L33), `TestS3FileCreateUncertainEffectLocksNativeBoxAndNeverReplays` (L72), `TestS3FileCreateVerifiedNoEffectDoesNotLockBox` (L104), `TestS3FileCreateAmbiguousWorkerEffectRemainsLocked` (L148)

## internal/state/h4b_capacity_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestH4BScopedIdentityReserveAndReplay` (L12), `TestH4BPeerCapacityReportsRetainedAndUnknown` (L76)

## internal/state/lifecycle_atomic_s4_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4CreateUUIDOriginalTerminalAtomicRollbackAndSuccess` (L29), `TestS4DeleteTombstoneOriginalTerminalAtomicAndUnknownCrash` (L74), `TestS4DeleteAtomicTerminalTombstoneWorkerAndReplay` (L138)

## internal/state/lifecycle_recovery_pin_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestN3LifecycleRecoveryPin` (L9)

## internal/state/lifecycle_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS2NameResourceLockBeforeNativeUUID` (L12), `TestS2QueuedCancelIsAtomicAndNoNativeEffect` (L57), `TestS2CreatedNativeUUIDAndOriginalOperationBindAtomically` (L88), `TestS2NativeRecordedOutputNeverDefaultsGuestExit` (L132), `TestS2RecoverUndispatchedDoesNotReplay` (L164), `TestS2BoundCreateBeforeCompletionRejectsDifferentKeyDeleteAdmits` (L195)

## internal/state/maintenance_observe_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestH1AMaintenanceRetainedAndUnknown` (L10), `TestH1AMaintenanceForeignAndRedaction` (L45), `TestH1BMaintenanceHistoryWitness` (L77)

## internal/state/maintenance_offline_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestH2BExactOriginalObservation` (L10)

## internal/state/migration_history_s1_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS1ForeignSQLiteMigrationHistoryFailClosed` (L18), `TestS1ConcurrentSQLiteOpenAndOriginalOperation` (L89), `TestS1AbortedSQLiteMigrationTransaction` (L154)

## internal/state/network_audit_ledger_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B2CompleteNetworkAuditLedger` (L35), `TestS4B2NetworkAuditLedgerRefusesDrift` (L57), `TestCompleteNetworkAuditLedger1024Boundary` (L86)

## internal/state/network_audit_read_only_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B2ReadOnlyAuditRefusesMissingUnsafeDB` (L12), `TestS4B2ReadOnlyAuditSeesLiveWALAndRefusesSchemaDrift` (L91)

## internal/state/network_claims_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B2NetworkMultiResourceAdmissionAndUnknownFence` (L10), `TestS4B2NetworkClaimGuardRejectsUnboundedAndBadKeys` (L75)

## internal/state/network_create_binding_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestVerifiedCreateBindingAtomic` (L9), `TestVerifiedCreateBindingRejectsInvalid` (L78), `TestCreatePinOriginalCollisionAndUnknown` (L92), `TestCreatePinLifecycleRetainsFences` (L148)

## internal/state/network_create_recovery_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestN3StagedCreateRecoveryAtomic` (L9), `TestN3LostCreateACKRetirementLineage` (L101)

## internal/state/network_delete_atomic_n3_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestN3ModernDeleteAtomicRecoveryFenceFault` (L9)

## internal/state/network_delete_recovery_n3_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestN3ModernDeleteNeverDowngradesMissingPin` (L39), `TestN3ModernDeleteRejectsDamagedCreateProvenance` (L78), `TestN3DeleteRequiresLostCreateACKExactRecoveryLineage` (L108)

## internal/state/network_deleted_binding_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestNetworkHistoryBeyond256AndExactCorruption` (L42), `TestNetworkHistoryRejectsOutstandingDependencies` (L68)

## internal/state/network_fences_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B2AddressFenceNeverRecyclesAcrossUUIDIncarnation` (L12), `TestS4B2AddressFenceAndLifecycleSurviveReopen` (L67)

## internal/state/network_forward_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestB4ForwardAtomicAdmissionConfirmDeleteAndUnknownFence` (L17), `TestB4ForwardQueuedCancellationAndRestartRelease` (L139), `TestForwardWildcardReservation` (L227)

## internal/state/network_ipv4_compaction_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestIPv4PeerCompactionDurableFenceAndHistory` (L125)

## internal/state/network_ipv4_retirement_churn_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestIPv4RetirementQueuePersistentRoundRobin` (L72), `TestIPv4RetirementChurnBeyond256Originals` (L120), `TestIPv4RetirementStatusScopedDiagnostics` (L152)

## internal/state/network_ipv4_retirement_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestIPv4RetirementAtomicArchiveReuse` (L65), `TestIPv4RetirementRefusesStaleOrUnsettledEvidence` (L136), `TestIPv4RetirementConcurrentReplacementOneSlot` (L185), `TestIPv4RetirementHeldWriterReservation` (L241), `TestIPv4RetirementVerifiedForwardHistory` (L317)

## internal/state/network_lifecycle_guard_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B2NativeNetworkDependenciesBlockAllRecyclingLifecycle` (L11)

## internal/state/network_n4_capacity_perf_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestN4ProjectCapacityAndNetworkAudit1024LocalBudget` (L17)

## internal/state/network_native_inventory_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestN3NativeLedgerFullScopeAndCorruption` (L11), `TestN3NativeAuditCursorAuthenticatedAndChangeRefusal` (L44), `TestN3NativeLedgerOriginalCreateJournalAndClaimsTamper` (L74), `TestN3NativeAuditManyIssuesRemainPageable` (L100), `TestN3NativeNetworkLedger1024ProjectLimitAndNoPartialReport` (L118)

## internal/state/network_peer_grant_snapshot_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B3ReadExactPeerCompilerSnapshot` (L9), `TestS4B3ReadPeerCompilerSnapshotFailsClosed` (L42), `TestS4B3ReadHistoricalRevokedSnapshotRequiresCurrentLease` (L75)

## internal/state/network_peer_grant_stage_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B3DurableGrantStageNoNativeEffect` (L42), `TestS4B3GrantStageFailsClosedOnActorAndNetworkDrift` (L66), `TestS4B3RegrantOnlyExactRevokedIncarnation` (L101)

## internal/state/network_peer_lineage_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPeerVerifiedGrantLinkIsAtomicAndNeverBackfilledOnUnknown` (L52), `TestPeerGrantRevokeRegrantVerifiedOriginalNeverAliasesPreviousOriginal` (L84), `TestPeerCompletedUnknownGrantReconciliationRetainsOriginalWithoutReplay` (L142), `TestPeerCompletedUnknownRevocationReconciliationRetainsOriginal` (L164)

## internal/state/network_peer_operation_mode_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B3PeerOriginalModeSelectorNeverAuthorizesWrongBranch` (L9)

## internal/state/network_peer_project_barrier_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B2RootAllSiblingProjectAdmissionBarrier` (L10), `TestS4B2RootStageAndReadRecheckOldSiblingOperations` (L97)

## internal/state/network_peer_replay_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B3PeerIdempotencyRestoresOriginalWithoutNativeLease` (L13)

## internal/state/network_peer_revoke_snapshot_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B3RevokeOneDirectionPreservesOtherConfirmed` (L20), `TestS4B3RevokeSnapshotRejectsUnknownOrStaleSibling` (L39)

## internal/state/network_peer_root_grant_recovery_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B3CompletedGrantLostACKReadOnlyRecoveryNoReplay` (L22), `TestS4B3IncompleteGrantRecoveryRetainsUNKNOWN` (L65)

## internal/state/network_peer_root_grant_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B3RootGrantOnlyExactClaimedOriginal` (L31), `TestS4B3RootGrantRejectsPoststageDrift` (L54)

## internal/state/network_peer_root_grant_worker_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B3GrantAtomicFinalizeOnlyVerifiedOriginal` (L9), `TestS4B3GrantFinalizeRefusesAmbiguousLedger` (L43)

## internal/state/network_peer_root_h3b_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestH3BPreparedRootDurableStageAndNoReplay` (L10)

## internal/state/network_peer_root_handoff_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B2RootRevocationOriginalOnlyAndClaims` (L70), `TestS4B2RootRevocationCannotBypassClaimsOrUNKNOWN` (L98)

## internal/state/network_peer_root_installed_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B2InstalledRootControllerExactOperationVerifiedWithoutCallbacks` (L10), `TestS4B2InstalledRootControllerRejectsUnverifiedKernelAndNeverReplays` (L49)

## internal/state/network_peer_root_recovery_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B2RootUnknownCompletedReceiptReadOnlyRecoveryAndNewAdmission` (L11), `TestS4B2RootUnknownIncompleteOrForgedReceiptPreservesFence` (L60), `TestS4B2RootUnknownGenericLegacyRecoveryCannotReleasePeerClaims` (L125)

## internal/state/network_peer_root_stage_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B2ControllerStagesOriginalPeerRevocation` (L16), `TestS4B2ControllerCannotStageInvalidPeerGrant` (L43)

## internal/state/network_peer_root_worker_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B2ControllerRootRevocationAtomicVerifiedFinish` (L37), `TestS4B2ControllerRootRevocationUnknownFences` (L82)

## internal/state/network_peer_scope_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4B3ResolvePeerScopeOnlyPinnedCurrentOwnedPair` (L8)

## internal/state/network_peer_scoped_admission_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR178N3InstalledAdmissionReplayAndRecovery` (L10)

## internal/state/network_peer_scoped_capacity_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR178N5PolicyHistoryCapacity` (L27)

## internal/state/network_peer_scoped_finalize_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR178N3AtomicScopedPeerFinalization` (L26)

## internal/state/network_peer_scoped_lineage_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR178V2ScopedRootRefusesForgedLegacyPeerHistory` (L13), `TestPR178V2StageRefusesForgedLegacyPeerHistory` (L51), `TestPR178V2RegrantRefusesForgedCompletedRevokeLineage` (L75), `TestPR178V2OldCredentialRevocationAfterSuccessKeepsHistoricalProof` (L96), `TestPR178V2UnknownHistoryDoesNotPermitNativeRecovery` (L117)

## internal/state/network_peer_scoped_original_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR178ScopedOriginalReadOnlyDirectionGrantAndRevoke` (L90), `TestPR178ScopedOriginalUNKNOWNReadOnlyNeverDispatch` (L151), `TestPR178ScopedOriginalFailsClosedForOriginalAndLedgerDrift` (L168), `TestPR178ScopedOriginalOtherActivePeerAndCancelRefused` (L210)

## internal/state/network_peer_scoped_stage_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR178V2StageScopedPeerGrantAndRevokeAtomic` (L16), `TestPR178V2StageScopedPeerRegrantOnlyExactTombstone` (L56), `TestPR178V2StageScopedPeerRejectsUnsafeLedgerAndOriginal` (L75), `TestPR178V2StageScopedPeerCancelledAndRevokeMissingRejected` (L116)

## internal/state/network_peer_scoped_terminal_policy_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR178N3LifecycleWithTerminalPeerPolicy` (L13)

## internal/state/network_pin_repair_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestN3RepairMissingLedgerAtomicAndReplay` (L45), `TestN3RepairDoesNotChangeConflictingRecords` (L132), `TestN3ModernNetworkIssuesRecoveryNavigation` (L178), `TestN3ConcurrentPinRepairOnlyOneOriginalWriter` (L215)

## internal/state/network_recovery_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS5A4ForwardRecoveryExactClaimsAndAtomicReceipt` (L87), `TestS5A4ForwardMissingAckOrDriftKeepsListener` (L141), `TestS5A4DependentDeleteLinkedChildrenAndFinalRecovery` (L189), `TestS5A4PeerRootCompletedGrantRevokeAtomic` (L293)

## internal/state/network_scoped_audit_ledger_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR178ScopedAuditLedgerReadOnlyExactPinnedPerBoxACL` (L29), `TestPR178ScopedAuditLedgerRejectsPartialMigrationAndNativeIdentityDrift` (L61), `TestPR178ScopedAuditLedgerRefusesUntrustedRequests` (L96), `TestPR178ScopedAuditLedgerSQLNeverAdmitsDuplicateBridgeAddress` (L113), `TestPR178ScopedAuditLedger1024SQLiteCapacity` (L130)

## internal/state/network_scoped_create_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR178ScopedCreateAtomicACLExchange` (L13)

## internal/state/network_scoped_retired_history_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR178N3ScopedRetiredHistory` (L76)

## internal/state/network_scoped_retirement_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestScopedRetirementLedgerEmptyAndLiveRefused` (L13), `TestScopedRetirementLedgerRetainsTerminalClaims` (L47)

## internal/state/oauth_actor_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4OAuthGrantActorReplayAfterTerminalAndCredentialIsolation` (L12), `TestS4OAuthRefreshQueuedClaimAndForeignActor` (L78)

## internal/state/oauth_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestOAuthAtomicPKCERefreshAndOriginRevocation` (L42), `TestOAuthConcurrentRefreshSingleSuccess` (L127), `TestOAuthFailedSQLiteTransactionDoesNotConsumeCodeOrRefresh` (L178), `TestOAuthRejectUntrustedOriginRedirectAndPKCE` (L236), `TestOAuthCredentialAndTunnelOriginResetInvalidatesPendingAndGrants` (L259), `TestOAuthAccessIDRevokeAlsoRevokesRefreshFamily` (L303), `TestOAuthRestartPersistenceAndLocalRevoke` (L348), `TestOAuthLocalRevokeSQLiteFaultRollsBackAccessAndGrant` (L425), `TestOAuthClientRedirectAudienceExpiryNegatives` (L461), `TestOAuthReplayedRefreshAfterRestartRevokesActiveGrant` (L501), `TestOAuthReplayRevocationSQLiteFaultIsAtomic` (L575)

## internal/state/operation_query_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS2CorrelationOriginalReceiptAndPrivateCursor` (L15), `TestS2FreshSchemaCrashUnknownAndLabelsPersist` (L136), `TestS2OperationCursorPreservesFirstPageAdmissionWatermarkAcrossWriter` (L197), `TestS2CursorStatusChangeDuringPagingKeepsIdentityAndFilter` (L314)

## internal/state/operation_recover_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS5A2RecoveryAtomicReceiptAndReplay` (L56), `TestS5A2RecoveryCommitFaultRollsBackBoth` (L119), `TestS5A2RecoveryConcurrentSameKeyAndDifferentKey` (L144), `TestS5A2RecoveryBindDoesNotUnlockOrInventNativeAck` (L186), `TestS5A2RecoveryExecEvidenceAndDeleteAtomic` (L228), `TestS5A2RecoveryRefusalAndLocalParity` (L266)

## internal/state/process_crash_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestProcessCrashUnknownNoReplay` (L17)

## internal/state/project_capacity_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestProjectCapacityConcurrentAdmissionAndReplay` (L24), `TestProjectCapacityUnknownCreateStillOccupiesSlot` (L91)

## internal/state/readiness_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS4CReadinessAdmissionFenceReplayRecovery` (L13), `TestS4CReadinessIntentBounds` (L129)

## internal/state/recovery_navigation_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS5A5RecoveryNavigationExactLinksAndBoundedLegacy` (L11), `TestS5A5RecoveryNavigationLimitsAndRedaction` (L63), `TestS5A5BoxIssuesIncludeSiblingProjectBarrier` (L88)

## internal/state/recovery_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestManualUnknownRecoveryNoReplayAndResourceLock` (L13), `TestManualRecoveryCannotAdoptAbsentOwnership` (L116)

## internal/state/replay_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS2ReplayAfterNativeDeleteOrUnknownWithoutReadback` (L10)

## internal/state/reviewed_recovery_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestPR177D4ReviewedCommitBoundary` (L24), `TestPR177D4ReviewedConcurrencyAndIdentityBinding` (L132), `TestPR177D4ExistingOnlyWriter` (L189)

## internal/state/schema_integrity_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestSameVersionSQLiteDDLDriftAndTriggersFailClosed` (L12)

## internal/state/scoped_drain_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestH2AScopedDrainActorNotBearer` (L11), `TestH2AScopedDrainWitnessRefusesUnknown` (L37)

## internal/state/store_alias_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS2SQLiteAliasAndHardlinkCannotBypassServeLock` (L14), `TestS2SQLiteTwoLocalWritersSerializeWithoutJournalDamage` (L76)

## internal/state/store_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestAdmissionIdempotencyAndRevocation` (L25), `TestCrashRecoveryNeverRedispatches` (L64), `TestPermissionsRejectUnsafeFile` (L100), `TestDatabaseSymlinkCannotAdoptOtherFile` (L126), `TestForeignDatabaseContentsAreNotModified` (L148)

## internal/state/worker_bh018_atomic_crash_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestBH018AtomicFinalizationRollsBackEverythingOnEventInsertFailure` (L72), `TestBH018AtomicFinalizationSurvivesDatabaseReopenAndRecovery` (L113), `TestBH018AtomicFinalizationRejectsForeignOrMalformedProvenance` (L164)

## internal/state/worker_bh018_atomic_finalization_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestBH018AtomicFinalizationNoPartialReceiptOnSQLiteAbort` (L49), `TestBH018AtomicFinalizationSurvivesStoreCloseReopen` (L97), `TestBH018AtomicFinalizationRejectsForeignIdentityWithoutAnyRow` (L156), `TestBH018WorkerRejectsForgedAtomicExecCompletionSignal` (L194), `TestBH018AtomicFinalizationRejectsOutOfRangeExitBeforeWrite` (L228)

## internal/state/worker_bh018_atomic_sigkill_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestBH018SIGKILLFinalizerChild` (L21), `TestBH018ActualSIGKILLBeforeAndAfterAtomicExecCommit` (L98)

## internal/state/worker_bh018_exec_boundary_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestBH018ExecFailureDispatchBoundaryPreservesOriginal` (L13)

## internal/state/worker_bh018_lost_finalize_ack_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestBH018WorkerLostFinalCommitAcknowledgementCannotOverwriteTerminal` (L15)

## internal/state/worker_bh018_special_terminal_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestBH018ExactTerminalSpecialExitPersistsAndReleasesClaimWithoutReplay` (L19)

## internal/state/worker_fault_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestS2NativeAckEventRollbackRetainsOriginalUnknown` (L16), `TestS2SQLiteKilledDuringUncommittedNativeAck` (L97)

## internal/state/worker_n3_ack_loss_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestN3WorkerBothNativeACKWritesRejectedProtectsUnknown` (L12)

## internal/state/worker_verified_create_failure_test.go

P1/P3 C02/C05: schema, claims, original recovery.

`TestCreateNativeFailureRetainedOwnedStoppedBoxRequiresExplicitDelete` (L10), `TestCreateVerifiedTerminalFailureVersusAmbiguousOutcome` (L63)

## Interpretation

All test bodies and dynamic subtests remain in the exact baseline tree. P1-P4 must record per affected assertion disposition, current replacement witness, and actual Box/CI result before any removal. Do not treat a name or a source-format gate as a tested capability.

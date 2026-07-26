# Pipeline v2 rollout

## Required production configuration

- Use coordinated versions of FastAPI, Node, Go and the frontend.
- Set `APP_ENV=production`, `PIPELINE_V2_ENABLED=true` and
  `PIPELINE_NAMESPACE=stratai:v2`.
- Configure three different random secrets of at least 32 characters:
  `SESSION_SECRET_KEY`, `INTERNAL_SERVICE_SECRET` and
  `CREDENTIAL_ENCRYPTION_KEY`.
- Set `PUBLIC_BACKEND_URL`, `FRONTEND_URL`, `TRUSTED_HOSTS` and
  `FORWARDED_ALLOW_IPS` explicitly.
- Keep Node and Go outside ingress. Expose the Go admin address only to the
  metrics collector.
- Use persistent Redis with ACLs and TLS. Do not delete legacy keys during the
  rollout.

## Baseline

1. Deploy metrics with `PIPELINE_V2_ENABLED=false`.
2. Process at least 30 representative demos on one Go replica.
3. Capture CPU and allocation profiles from the administrative pprof port:

   ```text
   /debug/pprof/profile?seconds=30
   /debug/pprof/allocs
   ```

4. Record p95 analysis duration, CPU-seconds per demo and peak RSS.

## Canary

1. Enable v2 for one coordinated canary deployment.
2. Confirm that the BullMQ scheduler list contains one `match-discovery`
   scheduler.
3. Validate OpenID, onboarding with zero matches, a real download, analysis,
   SSE reconnection and an intentional retry.
4. Expand gradually while comparing the canary with the baseline.

Rollback if the failure rate rises by more than one percentage point, p95
latency rises by more than 10%, or memory exceeds its limit. Retain the
`stratai:v2` keys for diagnosis and deploy the prior coordinated version.

## Optimization acceptance gate

Accept parser optimizations only when the 30-demo benchmark reduces p95
analysis time or CPU-seconds per demo by at least 30%, while peak memory grows
by no more than 10%. Evaluate PGO only after code-level hotspots have
stabilized.

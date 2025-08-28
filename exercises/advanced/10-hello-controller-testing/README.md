# Exercise

Write unit tests for the operator we built. Test the following:

1. `Reconcile`:
   - `ServiceDeployment` not found -> cleanup service and deployment.
   - `CreateOrUpdate` `Deployment` succeeds.
   - Conflict on `Deployment` update.
   - Error applying `Service`.

**Dependencies**

Use the following packages:

- https://github.com/stretchr/testify
  - https://github.com/stretchr/testify/assert
  - https://github.com/stretchr/testify/suite
  - https://github.com/stretchr/testify/require

For Mocking use:

- https://github.com/ovechkin-dm/mockio ([v1.0.0](https://ovechkin-dm.github.io/mockio/v1.0.0/))

---

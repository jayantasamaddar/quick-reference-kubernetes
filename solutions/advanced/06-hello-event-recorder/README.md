# About

If you look at the current deployed ServiceDeployment (CR), we will notice that `Events: <none>`.

![Service Deployment | CR | Events - None](../../assets/servicedeployment-crds-events-none.png)

This is despite we had the following events:

1. During creation: `Create Deployment "nginx"`, `Create Service "nginx-svc"`.
2. During modification: `Modified Deployment "nginx"`, `Modified Service "nginx-svc"`.
3. Detection of drift and self-healing: `Manual <Resource> deletion drift detected, self-heal in Progress`, `<Resource> self-healed as per ServiceDeployment`.

etc.

In other words, we are not recording events.
`k8s.io/client-go/tools/record` is the Kubernetes event recording subsystem.

When you run `kubectl describe <pod|deployment|crd>`, you often see an Events section like:

```
Events:
  Type    Reason   Age   From               Message
  ----    ------   ----  ----               -------
  Normal  Created  1s    my-operator        Created Deployment myapp
  Warning Failed   5s    my-operator        Failed to update Service
```

Those lines are Events stored in the events.k8s.io API.
The record package is how controllers/operators create those events.

The goal of this exercise is to use `recorder.EventRecorder` from `k8s.io/client-go/tools/record` package to record events.

---

# Exercise

Ensure the following events are recorded:

1. During creation: `Create Deployment "nginx"`, `Create Service "nginx-svc"`.
2. During modification: `Modified Deployment "nginx"`, `Modified Service "nginx-svc"`.
3. Detection of drift and self-healing: `Manual <Resource> deletion drift detected, self-heal in Progress`, `<Resource> self-healed as per ServiceDeployment`.
4. During deletion: `Deleted deployment "nginx"` | `Deleted service "nginx-svc"` (although short lived, as once `ServiceDeployment` is deleted this information would be lost).

5. **Bonus**: Ensure any benign race conditions do not throw Warning Events.

   ![Service Deployment | CR | Events - Benign Race Condition](../../assets/servicedeployment-crds-events-benign-race.png)

   The above is a write-conflict (409 Conflict) on the Deployment. It’s benign and very common right after you create a Deployment.

   Why it happens?

   Between your `Get` and `Update`/`Patch` inside `CreateOrUpdate`, some other actor updates the `Deployment` and bumps its `resourceVersion`:

   - the API server’s **defaulting** (fills defaults),
   - the **Deployment controller** (adds/updates fields, managedFields),
   - an **admission webhook** (if any),
   - or even another instance of your controller (if accidentally running twice).

   That makes your patch’s `resourceVersion` stale -> one attempt fails with conflict. Your next reconcile (or a retry) gets the latest object and applies successfully — hence the subsequent `DeploymentUpdated` events.

   The course of action is to treat these conflicts as benign and not show warnings. hey’re expected and self-heal. Only warn for "real" errors.

   Ideally, this is how the solution should look:

   ![Service Deployment | CR | Events - Benign Race Condition handled gracefully](../../assets/servicedeployments-crds-events-benign-race-handled-gracefully.png)

---

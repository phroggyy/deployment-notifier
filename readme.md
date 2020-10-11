# Deployment Notifier

The deployment notifier runs a WebSocket server that will notify
clients of when a deployment (determined by its label) is updated.

The output includes a version (the Kubernetes revision), and availability
expressed as `readyReplicas / desiredReplicas`.

## Intended Use

The primary use case for this is a client-side application with
asynchronous chunked module loading, where a module's source location
may change when a new release is rolled out. By having the client
connected to this WebSocket server, it can automatically (or ideally on page navigation)
hard refresh the page once availability is `1` in order to get the
latest bundle. 

## Sample Output

```
{"event":"updated","version":5,"availability":0.16666666666666666}
{"event":"updated","version":5,"availability":0.3333333333333333}
```

## Limitations

Currently, the notifier relies on labels to match on both
the ReplicaSets and the Deployment. In the future, this would ideally
rely on the `ownerReference` on the ReplicaSet.

# Bouquet

Bouquet is a draft addon manager for the Gardener.
It incorporates some of the requested features of the community but not
yet all of them.

> Caution: This software is early alpha. It is not meant for production use
> and shall (currently) only serve as a possible outlook of what is possible
> with pre-deployed software on Gardener Kubernetes clusters.

## Installation

If you want to deploy Bouquet on a target Gardener cluster, run the following:

```bash
helm install charts/bouquet \
  --name gardener-bouquet \
  --namespace garden
```

This will deploy Bouquet with the required permissions into your garden
cluster.

## Example use cases

As of now, Bouquet comes with two new custom resources: `AddonManifest` and
`AddonInstance`.

An `AddonManifest` can be considered equivalent to a Helm template. The
manifest itself only contains metadata (like the name, default values etc.).
The actual content of a manifest is specified via its `source` attribute.
Currently, the only available source is a `ConfigMap`.

An `AddonInstance` references an `AddonManifest` and a target `Shoot`. It
may also contain value overrides in its spec. As soon as an `AddonInstance`
is created, Bouquet will apply the values to the templates and then ensure
that the objects exist in the target shoot.
If an `AddonInstance` is deleted, Bouquet will also make sure that the
created objects are deleted as well.

Since this is just a tech-preview, features like value / chart updates, more
efficient templating, company addon guidelines etc. are not yet implemented /
yet to come / yet to be discussed.


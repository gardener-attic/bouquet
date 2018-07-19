# The Chart Repository Guide

## Prerequistes
* Go through the [Quickstart](quickstart.md) Guide
* Read through the [Charts](charts.md) document

## Create a chart repository
A _chart repository_ is an HTTP server that houses one or more packaged charts. When you're ready to share your charts, the preferred mechanism is a chart repository. You can contribute to the official helm chart repository or create your own. Here we'll talk about creating your own chart repository.

Because a chart repository can be any HTTP server that can serve YAML and tar files and can answer GET requests, you have a plethora of options when it comes down to hosting your own chart repository. For example, you can use a Google Cloud Storage(GCS) bucket, Amazon S3 bucket, or even create your own web server.

### The chart repository structure
A chart repository consists of packaged charts and a special file called `index.yaml` which contains an index of all of the charts in the repository. A chart repository has a flat structure. Given a repository URL, you should be able to download a chart via a GET request to `URL/chartname-version.tgz`. 

For example, if a repository lives at the URL: `http://helm-charts.com`, the `alpine-0.1.0` chart would live at `http://helm-charts.com/alpine-0.1.0.tgz`. The index file would also live in the same chart repository at `http://helm-charts.com/index.yaml`.

#### The index file
The index file is a yaml file called `index.yaml`. It contains some metadata about the package as well as a dump of the Chart.yaml file of a packaged chart. A valid chart repository must have an index file. The index file contains information about each chart in the chart repository. The `helm repo index` command will generate an index file based on a given local directory that contains packaged charts.

This is an example of an index file:
```
alpine-0.1.0:
  name: alpine
  url: http://storage.googleapis.com/kubernetes-charts/alpine-0.1.0.tgz
  created: 2016-05-26 11:23:44.086354411 +0000 UTC
  checksum: a61575c2d3160e5e39abf2a5ec984d6119404b18
  chartfile:
    name: alpine
    description: Deploy a basic Alpine Linux pod
    version: 0.1.0
    home: https://github.com/example-charts/alpine
redis-2.0.0:
  name: redis
  url: http://storage.googleapis.com/kubernetes-charts/redis-2.0.0.tgz
  created: 2016-05-26 11:23:44.087939192 +0000 UTC
  checksum: 2cea3048cf85d588204e1b1cc0674472b4517919
  chartfile:
    name: redis
    description: Port of the replicatedservice template from kubernetes/charts
    version: 2.0.0
    home: https://github.com/example-charts/redis
```

We will go through a detailed GCS example here, but feel free to skip to the next section if you've already created a chart repository.

The first step will be to **create your GCS bucket**. We'll call ours `fantastic-charts`.

![Create a GCS Bucket](images/create-a-bucket.png)

Next, you'll want to make your bucket public, so you'll want to **edit the bucket permissions**.

![Edit Permissions](images/edit-permissions.png)

Insert this line item to **make your bucket public**:

![Make Bucket Public](images/make-bucket-public.png)

Congratulations, now you have an empty GCS bucket ready to serve charts!

## Store charts in your chart repository
Now that you have a chart repository, let's upload a chart and an index file to the repository.
Charts in a chart repository must be packaged (`helm package chart-name/`) and versioned correctly (following [SemVer 2](http://semver.org/) guidelines).

These next steps compose an example workflow, but you are welcome to use whatever workflow you fancy for storing and updating charts in your chart repository.

Once you have a packaged chart ready, create a new directory, and move your packaged chart to that directory.

```console
$ helm package docs/examples/alpine/
Archived alpine-0.1.0.tgz
$ mkdir fantastic-charts
$ mv alpine-0.1.0.tgz fantastic-charts/
```

Outside of your directory, run the `helm repo index [DIR] [URL]` command. This command takes the path of the local directory that you just created and the URL of your remote chart repository and composes an index.yaml file inside the given directory path.

```console
$ helm repo index fantastic-charts http://storage.googleapis.com/fantastic-charts
```

Now, you can upload the chart and the index file to your chart repository using a sync tool or manually. If you're using Google Cloud Storage, check out this [example workflow](chart_repository_sync_example.md) using the gsutil client.

## Add a new chart to your chart repository

When you've created another chart, move the new packaged chart into the fantastic-charts directory, and run the `helm repo index [DIR] [URL]` command once again to update index.yaml. Then, upload the index.yaml file and the new chart to the repository.

## Share your charts with others
When you're ready to share your charts, simply let someone know what the url of your repository is.

*Note: A public GCS bucket can be accessed via simple http at this address `http://storage.googleapis.com/bucket-name`.*

From there, they will add the repository to their helm client via the `helm repo add [NAME] [URL]` command with any name they would like to use to reference the repository.

```console
$ helm repo add fantastic-charts http://storage.googleapis.com/fantastic-charts
$ helm repo list
fantastic-charts    http://storage.googleapis.com/fantastic-charts
```

*Note: A repository will not be added if it does not contain a valid index.yaml.*

After that, they'll be able to search through your charts. After you've updated the repository, they can use the `helm update` command to get the latest chart information.

*Under the hood, the `helm repo add` and `helm update` commands are fetching the index.yaml file and storing them in the `$HELM_HOME/repository/cache/` directory. This is where the `helm search` function finds information about charts.*

## Contributing charts to the official helm chart repository
*Coming Soon*

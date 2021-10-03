# gdu - disk usage in go

A silly, just-for-learning, `du` clone in Golang.

This is my first experience with Go, and this is my little learning experiment:
building a `du` (disk usage) clone.

Do NOT blame me for any animals harmed by this tool!

This tool uses all available CPUs to parallel processing all dirs recursively.


## Build

```
$ go build
```

## Usage

```
$ ./gdu [-s] [-h] [path [...]]
```

## Help

```
$ ./gdu --help
```

## Go subroutine architecture

The main idea is to process every single dir (recursively) by a worker thread (or go routine), but only span as much go workers
as there are available CPUs.

The problem lies in the recursive processing: Each worker does not only GET jobs to process, but can also GENERATE new jobs.
This makes queuing with Go channels a bit tricky, as they are of limited size, and they block if they are full.

This leads to deadlocks if not handled properly. So I came up with the following process:

1. A "Job" is a file or directory that need to be inspected. Goal of the job is to read its file size and to generate
	new jobs for childs.
2. Use a `sync.WaitGroup` that counts Jobs, not Workers: Each job increases the WaitGroup's counter, while finishing one
	decreases it.
3. A single-item `channel` is used as job queue:
   1. Jobs are put into the channel by the main thread AND worker threads, using a non-blocking channel write
   2. As soon as a Worker has done its work, it decreases the WaitGroup, and fetches the next from the channel
   3. A worker puts a new Job on the channel for each child entry of an examined directory, but only if the
		channel is free at the moment (could be written). If the channel would block, the job is done by the actual worker.
4. The main thread spans the workers and enqueues the initial Jobs to the (blocking) channel, also working on Jobs if all workers
	are blocked at the moment. So also the main thread works :-)
5. As soon as the Main thread has generated all initial jobs, it waits for the `WaitGroup` to become empty / zero.
6. It informs the workers by closing the channels to finish them.

This makes processing also of big / deep directories very fast.



(c) 2021 Alex Schenkel
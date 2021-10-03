package gdu

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

func CreateFileLike(path string) (Filelike, error) {
	var ret Filelike

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fileInfo.IsDir() {
		dir := NewDir(path, fileInfo)
		ret = &dir
	} else if fileInfo.Mode().IsRegular() {
		file := NewFile(path, fileInfo)
		ret = &file
	}

	return ret, err
}

type JobQueue struct {
	WaitGroup *sync.WaitGroup
	JobQueue  chan Filelike
}

func (q *JobQueue) Join() {
	q.WaitGroup.Wait()
	close(q.JobQueue)
}

func (q *JobQueue) EnqueueJob(item Filelike) {
	q.WaitGroup.Add(1)
	select {
	case q.JobQueue <- item: // ok, someone else took it
	default:
		// do it myself, no one else has time:
		q.processJob(item)
	}
}

func (q *JobQueue) examineFilelike(file Filelike) {
	switch file.(type) {
	case *Dir:
		q.examineDir(file.(*Dir))
	case *File:
		q.examineFile(file.(*File))
	}
}

func (q *JobQueue) examineDir(dir *Dir) error {
	files, err := ioutil.ReadDir(dir.RelPath)
	if err != nil {
		return err
	}
	for _, file := range files {
		filelike, err := CreateFileLike(filepath.Join(dir.RelPath, file.Name()))
		if err == nil && filelike != nil {
			dir.Children = append(dir.Children, filelike)
			// enqueue for later, parallel examination:
			q.EnqueueJob(filelike)
		}
	}
	return nil
}

func (q *JobQueue) examineFile(file *File) {
	file.SizeBytes = uint64(file.FileInfo.Size())
}

func (q *JobQueue) processJob(item Filelike) {
	q.examineFilelike(item)
	q.WaitGroup.Done()
}

type Worker struct {
	queue *JobQueue
}

func NewWorker(queue *JobQueue) Worker {
	return Worker{
		queue,
	}
}

func (w *Worker) ProcessJobs() {
	for job := range w.queue.JobQueue {
		w.queue.processJob(job)
	}
}

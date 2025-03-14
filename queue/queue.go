package queue

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// struct to define a job
type Job struct {
	user   string // user IP
	Folder string // job folder
	status string // job status
}

// struct to define a queue
type Queue struct {
	nprocs      int        // max. number of concurrent goroutines
	expire_time float32    // time after which a non-retrieved process is discarded
	mutex       sync.Mutex // mutex to avoid race conditions
	jobs        []Job      // job list
}

// Create job from a given user tag
func MakeJob(user string) Job {
	base_dir, err := os.Getwd()
	if err != nil {
		log.Fatal("Unable to get CWD. The world is ending")
	}
	tdir, err := os.MkdirTemp(base_dir, "temp-queueinator-job")
	if err != nil {
		log.Fatal("Unable to create temporary directory ", tdir, " with error ", err)
	}

	j := Job{
		user:   user,
		Folder: tdir,
		status: "uploading",
	}

	return j
}

// Instantiate new queue and return pointer to it.
// Receives max. number of processes simultaneously running.
func NewQueue(nprocs int, expire_time float32) *Queue {
	var qu Queue

	qu.nprocs = nprocs
	qu.expire_time = expire_time

	qu.jobs = make([]Job, 0)

	return &qu
}

// "Tidy" queue by ordering jobs alternating users.
// Assumes the queue's mutex is locked
func (qu *Queue) tidyQueue() {
	job_map := make(map[string][]Job)
	users := make([]string, 0)

	for _, job := range qu.jobs {
		v, found := job_map[job.user]

		if !found {
			v = make([]Job, 0)
			job_map[job.user] = v
			users = append(users, job.user)
		}

		job_map[job.user] = append(v, job)
	}

	new_jobs := make([]Job, 0)

	for len(new_jobs) < len(qu.jobs) {
		for _, user := range users {
			v := job_map[user]

			if len(v) > 0 {
				new_jobs = append(new_jobs, v[0])
				job_map[user] = v[1:]
			}
		}
	}

	qu.jobs = new_jobs
}

// Submit new job to queue. Returns pointer to job
func (qu *Queue) SubmitJob(user string) *Job {
	qu.mutex.Lock()
	defer qu.mutex.Unlock()

	qu.tidyQueue()

	qu.jobs = append(qu.jobs, MakeJob(user))

	return &(qu.jobs[len(qu.jobs)-1])
}

// Check job status
func (qu *Queue) CheckJob(job_folder string) string {
	qu.mutex.Lock()
	defer qu.mutex.Unlock()

	for _, job := range qu.jobs {
		if job.Folder == job_folder {
			return job.status
		}
	}

	return "none"
}

// Set job status to a given value
func (qu *Queue) SetStatus(job_folder string, status string) {
	qu.mutex.Lock()
	defer qu.mutex.Unlock()

	for i, job := range qu.jobs {
		if job.Folder == job_folder {
			qu.jobs[i].status = status
			return
		}
	}
}

// Count number of running jobs.
func (qu *Queue) nRunningJobs() int {
	njobs := 0
	for _, job := range qu.jobs {
		if job.status == "running" {
			njobs++
		}
	}

	return njobs
}

// Get folder for next job from queue.
// Awaits until there are running slots available
// if there are already qu.nprocs goroutines running
func (qu *Queue) fetchNextJob(period float32) string {
	qu.mutex.Lock()
	njobs := qu.nRunningJobs()
	for njobs >= qu.nprocs {
		qu.mutex.Unlock()

		time.Sleep(time.Duration(period) * time.Second)

		qu.mutex.Lock()
		njobs = qu.nRunningJobs()
	}

	first_in_queue := ""
	for i, job := range qu.jobs {
		if job.status == "queue" {
			first_in_queue = job.Folder
			qu.jobs[i].status = "running"
			break
		}
	}

	qu.mutex.Unlock()

	return first_in_queue
}

// Get next job to run when any is available. Waits 1 period if none are available.
func (qu *Queue) NextJob(period float32) string {
	job_folder := qu.fetchNextJob(period)
	for len(job_folder) == 0 {
		time.Sleep(time.Duration(period) * time.Second)

		job_folder = qu.fetchNextJob(period)
	}

	return job_folder
}

// Remove job from queue
func (qu *Queue) RemoveJob(job_folder string, kill_warn bool) {
	qu.mutex.Lock()
	defer qu.mutex.Unlock()

	for i, job := range qu.jobs {
		if job.Folder == job_folder {
			if kill_warn {
				slog.Warn("Killing job " + job_folder + " due to timeout")
			}

			qu.jobs = append(qu.jobs[:i], qu.jobs[(i+1):]...)

			err := os.RemoveAll(job_folder) // delete folders here
			if err != nil {
				slog.Warn(fmt.Sprintf("unable to delete folder %s: error %v", job_folder, err))
			}

			return
		}
	}
}

// Run job
func (qu *Queue) RunJob(command string, job_folder string, expiration float32) {
	// define launcher between OSs
	launcher := "bash"
	flag := "-c"
	system := runtime.GOOS
	switch system {
	case "darwin":
		launcher = "bash"
		flag = "-c"
	case "linux":
		launcher = "bash"
		flag = "-c"
	case "windows":
		launcher = "cmd"
		flag = "/c"
	default:
		log.Fatal("Unsupported platform " + system)
	}

	cmd := exec.Command(launcher, flag, command)
	cmd.Dir = job_folder

	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Warn(fmt.Sprintf("error %v when running job %s", err, job_folder))
	}

	logfile, err := os.Create(filepath.Join(job_folder, "queueinator.log"))
	if err != nil {
		slog.Warn(fmt.Sprintf("error %v when writing log for job %s", err, job_folder))
	}
	defer logfile.Close()

	_, err = logfile.Write(output)
	if err != nil {
		slog.Warn(fmt.Sprintf("error %v when writing log for job %s", err, job_folder))
	}

	fmt.Println(time.Now().UTC(), ": Finishing job "+job_folder)
	qu.SetStatus(job_folder, "done")

	// launch goroutine to remove job after the established expiration time
	go func() {
		time.Sleep(time.Duration(expiration) * time.Second)

		qu.RemoveJob(job_folder, true)
	}()
}

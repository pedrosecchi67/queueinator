package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"queueinator/folder_handler"
	"queueinator/queue"
	"queueinator/server_exec"
	"slices"
	"strings"
	"time"
)

func PrintHelp() {
	help_message := string(`
	queueinator - simple TCP server manager for task queues
	Copyright (C) 2025  Pedro de Almeida Secchi (secchi.pedro@protonmail.com)
	This program comes with ABSOLUTELY NO WARRANTY.
	This is free software, and you are welcome to redistribute it
	under certain conditions. Check LICENSE for details.

	Usage:

	server mode:
	queueinator serve COMMAND PORT [-t CHECK_PERIOD] [-n NPROCS] [-b BUFFER_SIZE]
		[-e EXPIRE_TIME]

	The server checks for new jobs once every CHECK_PERIOD seconds (def. 1.0).
	Up to NPROCS processes are ran in parallel (def. 1).
	Incoming data must be constrained to BUFFER_SIZE Mb (def. 10).
	The job expires and is deleted if the client does not fetch the resulting data after
	EXPIRE_TIME seconds (def. 3600.0).

	cleanup mode:
	queueinator cleanup

	In a server, removes all previous job folders.

	client mode:
	queueinator run IP PORT [-t CHECK_PERIOD] [-b BUFFER_SIZE]

	Sends contents of current folder to server at IP:PORT with data limit of BUFFER_SIZE Mb.
	Checks for process conclusion/fetches results once every CHECK_PERIOD seconds.
	`)

	fmt.Println(help_message)
}

func main() {

	allowed_kwargs := []string{
		"t",
		"n",
		"b",
		"e",
	}

	pos_args := make([]string, 0)
	kwargs := map[string]string{
		"t": "1.0",
		"n": "1",
		"b": "10",
		"e": "3600.0",
	}

	i := 1
	for i < len(os.Args) {
		arg := os.Args[i]

		if arg == "--help" || arg == "-h" {
			PrintHelp()
			return
		}

		if strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "--") {
			i++

			key := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")

			if !slices.Contains(allowed_kwargs, key) {
				PrintHelp()
				log.Fatal("Argument " + key + " unrecognized.")
			}

			kwargs[key] = os.Args[i]
		} else {
			pos_args = append(pos_args, arg)
		}

		i++
	}

	if len(pos_args) < 1 {
		PrintHelp()
		log.Fatal("Wrong argument structure.")
	}
	mode := pos_args[0]

	if mode == "serve" {
		if len(pos_args) != 3 {
			PrintHelp()
			log.Fatal("Wrong argument structure.")
		}

		command := pos_args[1]
		port := pos_args[2]

		var period float32
		var expire_time float32
		var nprocs int
		var buffer_size int

		// parse args
		_, err := fmt.Sscanf(kwargs["t"], "%f", &period)
		if err != nil {
			PrintHelp()
			log.Fatal("Input -t is inconsistent, expected float")
		}

		_, err = fmt.Sscanf(kwargs["e"], "%f", &expire_time)
		if err != nil {
			PrintHelp()
			log.Fatal("Input -e is inconsistent, expected float")
		}

		_, err = fmt.Sscanf(kwargs["n"], "%d", &nprocs)
		if err != nil {
			PrintHelp()
			log.Fatal("Input -n is inconsistent, expected int")
		}

		_, err = fmt.Sscanf(kwargs["b"], "%d", &buffer_size)
		buffer_size *= 1024 * 1024
		if err != nil {
			PrintHelp()
			log.Fatal("Input -b is inconsistent, expected int")
		}

		// output basic data
		fmt.Println("Running queue for command " + command + " at port " + port + ". Press Ctrl+C to interrupt.")
		fmt.Printf("Max. number of goroutines: %d\nChecking period: %f seconds\nBuffer size: %d Mb\nExpire time: %f seconds\n",
			nprocs, period, buffer_size/1024/1024, expire_time)

		// set up queue
		qu := queue.NewQueue(nprocs, expire_time)

		// set up goroutine that runs the queue
		go func() {
			fmt.Println("Fetching jobs...")
			for {
				job := qu.NextJob(period)
				fmt.Println(time.Now().UTC(), ": Running "+job)

				go qu.RunJob(command, job)
			}
		}()

		// run server
		listener, err := net.Listen("tcp", ":"+port)
		if err != nil {
			log.Fatal("Error establishing TCP connection: ", err)
		}
		defer listener.Close()

		for {
			conn, err := listener.Accept()
			if err != nil {
				conn.Close()
				log.Fatal("Error when accepting connection: ", err)
			}

			message := make([]byte, 0)
			buff := make([]byte, buffer_size)
			for {
				n, err := conn.Read(buff)
				if err != nil {
					if err == io.EOF {
						break
					}
					conn.Close()
					log.Fatal("Error when reading message: ", err)
				}
				message = append(message, buff[:n]...)
			}

			server_exec.MessageExecute(message, qu, conn, buffer_size)

			conn.Close()
		}
	} else if mode == "cleanup" {
		// delete temporary job folders
		folders, err := filepath.Glob("temp-queueinator-job*")
		if err != nil {
			log.Fatal("Unable to find temp-queueinator-job* files in current directory.")
		}

		for _, folder := range folders {
			fmt.Printf("Removing job folder %s...\n", folder)
			os.RemoveAll(folder)
		}
	} else if mode == "run" {
		if len(pos_args) != 3 {
			PrintHelp()
			log.Fatal("Wrong argument structure.")
		}

		ip_address := pos_args[1]
		port := pos_args[2]

		// parsing kwargs
		var period float32
		var buffer_size int

		_, err := fmt.Sscanf(kwargs["b"], "%d", &buffer_size)
		buffer_size *= 1024 * 1024
		if err != nil {
			PrintHelp()
			log.Fatal("Input -b is inconsistent, expected int")
		}

		_, err = fmt.Sscanf(kwargs["t"], "%f", &period)
		if err != nil {
			PrintHelp()
			log.Fatal("Input -t is inconsistent, expected float")
		}

		fmt.Printf("Running at %s:%s...\n", ip_address, port)

		// build message
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal("Impossible to read CWD! The end is nigh. ", err)
		}
		message, err := folder_handler.GenFolderMessage(cwd, buffer_size)
		if err != nil {
			log.Fatal("Unable to parse current folder contents: ", err)
		}

		// send message
		ret, err := server_exec.SendMessage(
			append([]byte("submit "), message...),
			ip_address, port, buffer_size)
		if err != nil {
			log.Fatal("Error while making submission: ", err)
		}

		job_tag := string(ret)

		// loop until the server is done running
		for {
			time.Sleep(time.Second * time.Duration(period))

			status, err := server_exec.CheckJobStatus(job_tag, ip_address, port)
			if err != nil {
				log.Fatal("Error while checking job status: ", err)
			}

			// act according to status
			if status == "none" {
				log.Fatal("Job " + job_tag + " no longer operational at server. Aborting")
			} else if status == "done" {
				// request content back from server
				message, err := server_exec.SendMessage(
					[]byte("retrieve "+job_tag),
					ip_address, port, buffer_size)
				if err != nil {
					log.Fatal("Error while retrieving job "+job_tag+": ", err)
				}

				// parse messsage to files and write here
				err = folder_handler.Parse2Files(cwd, message)
				if err != nil {
					log.Fatal("Error parsing retrieved job contents for "+job_tag+": ", err)
				}

				break // break client check loop
			} else if status == "uploading" || status == "queue" || status == "running" {
				continue
			} else {
				log.Fatal("Status message " +
					job_tag +
					" unknown. This is really weird. Aborting")
			}
		}
	} else {
		// I dont know what Im doing
		PrintHelp()
		log.Fatal("Unrecognized run mode " + mode)
	}

}

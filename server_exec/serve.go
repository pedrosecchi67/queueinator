package server_exec

import (
	"bytes"
	"fmt"
	"log/slog"
	"net"
	"os"
	"queueinator/folder_handler"
	"queueinator/queue"
	"strings"
)

// Execute tasks when receiving a given message.
// Takes reference to queue as an input.
func MessageExecute(message []byte, qu *queue.Queue, conn net.Conn, buffer_size int) {

	// cut header
	header_bytes, message, _ := bytes.Cut(message, []byte(" "))
	header := string(header_bytes)

	// parse message type
	if header == "submit" {
		ip_address, _, _ := strings.Cut(conn.RemoteAddr().String(), ":")

		// submit new job, create temp folder
		job := qu.SubmitJob(ip_address)
		err := folder_handler.Parse2Files(job.Folder, message)
		if err != nil {
			slog.Warn(
				fmt.Sprintf(
					"found error %v while parsing request from %s", err, ip_address))
		}
		// set status from uploading to queue
		qu.SetStatus(job.Folder, "queue")

		// send folder name to client
		conn.Write([]byte(job.Folder))
	} else if header == "check" {
		job_folder, _, _ := bytes.Cut(message, []byte(" "))
		status := qu.CheckJob(string(job_folder))

		// send status to client
		conn.Write([]byte(status))
	} else if header == "retrieve" {
		job_folder, _, _ := bytes.Cut(message, []byte(" "))

		// send all contents to client
		message, err := folder_handler.GenFolderMessage(string(job_folder), buffer_size)
		if err != nil {
			slog.Warn(
				fmt.Sprintf("found error %v while sending back request for job %s",
					err, string(job_folder)))
		}

		conn.Write(message)

		// remove job from queue and delete temp file
		qu.RemoveJob(string(job_folder))
		os.RemoveAll(string(job_folder))
	} else {
		slog.Warn(
			fmt.Sprintf(
				"could not understand header %s while parsing request", header))
	}

}

// Send message over TCP. Merely a utility function.
// Returns a byte array with the returned data and an error/nil
func SendMessage(message []byte, ip_address string, port string, buffer_size int) ([]byte, error) {
	conn, err := net.Dial("tcp", ip_address+":"+port)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	n, err := conn.Write(message)
	if err != nil {
		return nil, err
	}
	if n < len(message) {
		return nil, fmt.Errorf("unable to send all of the data via TCP")
	}
	// signal EOF
	conn.(*net.TCPConn).CloseWrite()

	buff := make([]byte, buffer_size)
	n, err = conn.Read(buff)
	if err != nil {
		return nil, nil
	}

	return buff[:n], nil
}

// Consult the server on the status of a job.
// Returns status string and error message
func CheckJobStatus(job_tag string, ip_address string, port string) (string, error) {
	status, err := SendMessage([]byte("check "+job_tag), ip_address, port, 1024)
	if err != nil {
		return "error", err
	}

	return string(status), nil
}

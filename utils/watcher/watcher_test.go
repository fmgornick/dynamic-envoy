package watcher

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestWatch(t *testing.T) {
	change := make(chan Message, 7)
	var msg Message

	go Watch(".", change)
	time.Sleep(time.Millisecond * 500)

	f, _ := os.Create("file.txt")
	time.Sleep(time.Millisecond * 10)
	f.WriteString("modification")
	time.Sleep(time.Millisecond * 10)
	os.Mkdir("directory", os.ModePerm)
	time.Sleep(time.Millisecond * 10)
	os.Rename("file.txt", "directory/file.txt")
	time.Sleep(time.Millisecond * 10)
	os.Remove("directory/file.txt")
	time.Sleep(time.Millisecond * 10)
	os.Remove("directory/")
	time.Sleep(time.Millisecond * 10)

	msg = <-change
	assert.Equal(t, Create, msg.Operation, "operation type should be Create (0)")
	assert.Equal(t, "file.txt", msg.Path, "path to file should be \"file.txt\"")

	msg = <-change
	assert.Equal(t, Modify, msg.Operation, "operation type should be Modify (1)")
	assert.Equal(t, "file.txt", msg.Path, "path to file should be \"file.txt\"")

	msg = <-change
	assert.Equal(t, Create, msg.Operation, "operation type should be Create (0)")
	assert.Equal(t, "directory", msg.Path, "path to file should be \"directory\"")

	msg = <-change
	assert.Equal(t, Create, msg.Operation, "operation type should be Create (0)")
	assert.Equal(t, "directory/file.txt", msg.Path, "path to file should be \"directory/file.txt\"")

	msg = <-change
	assert.Equal(t, Move, msg.Operation, "operation type should be Move (2)")
	assert.Equal(t, "file.txt", msg.Path, "path to file should be \"file.txt\"")

	msg = <-change
	assert.Equal(t, Delete, msg.Operation, "operation type should be Delete (3)")
	assert.Equal(t, "directory/file.txt", msg.Path, "path to file should be \"directory/file.txt\"")

	msg = <-change
	assert.Equal(t, Delete, msg.Operation, "operation type should be Delete (3)")
	assert.Equal(t, "directory", msg.Path, "path to file should be \"file.txt\"")
}

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"net/http"

	//	"github.com/djherbis/buffer"
	//	"github.com/djherbis/nio"
	"gopkg.in/pipe.v2"
)

const SOUND_COMMAND = "arecord -r 16000 -f S16_LE -c 1  -D pulse"
const PRECISE_COMMAND = "docker run   -a stdin -a stdout -i -v/home/ainu/v2j/model:/model literunner /litepre/wait-wake.py /model/model.tflite"
const FILTER_COMMAND = "./precise-filter-arm64   --trigger-level=5 --sensivity=0.6  --onetime-sensivity=0.6 --onetime-trigger-level=5"
const ASSISTANT_COMMAND = "php assistant.php"

var buf []byte

var RIFF []byte

var lastMinute []byte
var lastMinutePostion int = 0

func JustEcho() pipe.Pipe {
	return pipe.TaskFunc(func(s *pipe.State) error {
		/*
			buf = buffer.New(2048)

			nio.Copy(s.Stdout, s.Stdin, buf)
		*/
		//	buf := make([]byte, 0, 4096) // big buffer
		buf = make([]byte, 256)            // using small tmo buffer for demonstrating
		lastMinute = make([]byte, 3840000) // using small tmo buffer for demonstrating

		for {
			n, err := s.Stdin.Read(buf)
			if err != nil {
				if err != io.EOF {
					fmt.Println("read error:", err)
				}
				break
			}
			//	fmt.Println("readed: ", n)
			//	buf = append(buf, tmp[:n]...)
			written, _ := s.Stdout.Write(buf[:n])

			//fmt.Println("got", n, "bytes. Written ", written, " bytes,  arrsize: ", len(buf))
			_ = written
			lastMinutePostion = lastMinutePostion + n
			/*	if lastMinutePostion >= 3840000 {
				lastMinutePostion = 0
			}*/
			for i := 0; i <= n-1; i++ {
				if i+lastMinutePostion >= 3840000 {
					lastMinutePostion = 0
				}
				lastMinute[i+lastMinutePostion] = buf[i]
			}
		}

		return nil
	})
}

func handler(w http.ResponseWriter, r *http.Request) {
	currentTime := time.Now()
	lastMinuteCopy := make([]byte, 0, 3840000)
	lastMinuteCopy = append(lastMinuteCopy, RIFF...)
	lastMinuteCopy = append(lastMinuteCopy, lastMinute[:lastMinutePostion]...)
	lastMinuteCopy = append(lastMinuteCopy, lastMinute[lastMinutePostion:]...)
	//fmt.Fprintf(w, "%s", string(lastMinuteCopy))
	fmt.Fprintf(w, "OK")
	ioutil.WriteFile("lastminutes/"+currentTime.Format("2006-01-02 15.04.05")+".wav", lastMinuteCopy, 0777)

}

func main() {

	RIFF, _ = ioutil.ReadFile("empty.wav")
	fmt.Println("RIFF LEN:", len(RIFF))
	fmt.Println("Barrymore runner starting...")

	http.HandleFunc("/", handler)

	go http.ListenAndServe(":8080", nil)
	/*reader := new(bufio.Reader)
	writer := new(bufio.Writer)
	rwiter := bufio.NewReadWriter(reader, writer)*/
	p := pipe.Line(

		pipe.Exec("arecord", "-r", "16000", "-f", "S16_LE", "-c", "1", "-D", "pulse"),
		/*	pipe.Read(rwiter),
			//	pipe.Write(os.Stdout),
			pipe.Write(rwiter),*/
		JustEcho(),
		pipe.Exec("docker", "run", "-a", "stdin", "-a", "stdout", "-i", "-v/home/ainu/v2j/model:/model", "literunner", "/litepre/wait-wake.py", "/model/model.tflite"),
		pipe.Exec("./precise-filter-arm64", "--trigger-level=5", "--sensivity=0.6", "--onetime-sensivity=0.6", "--onetime-trigger-level=5"),
		pipe.Exec("php", "assistant.php"),
		pipe.Write(os.Stdout),
		//JustEcho(),
	)
	err := pipe.Run(p)
	if err != nil {
		fmt.Printf("ERROR! %v\n", err)
	}

}

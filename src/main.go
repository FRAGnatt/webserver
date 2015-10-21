package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	OK           string = "200 OK"
	NOT_FOUND    string = "404 NOT FOUND"
	ERROR        string = "500 INTERNAL SERVER ERROR"
	BAD_REQUEST  string = "400 BAD REQUEST"
	FORBIDDEN    string = "403 FORBIDDEN"
	NOT_ALLOWED  string = "405 METHOD NOT ALLOWED"
	DEFAULT_FILE string = "/index.html"
	FILE_404     string = "/404.html"
	HTTP_VERSION string = "1.1"
)

const (
	CONN_HOST string = "localhost"
	CONN_PORT string = "80"
	CONN_TYPE string = "tcp"
)

const (
	STATUS         string = "HTTP/1.1 %s\r\n"
	DATE           string = "Date: %s\r\n"
	CONTENT_TYPE   string = "Content-Type: %s\r\n"
	CONTENT_LENGTH string = "Content-Length: %d\r\n"
	SERVER         string = "Server: WebServerOnGo\r\n"
	CONNECTION     string = "Connection: close\r\n"
)

const (
	INDEX_FILE string = "index.html"
)

var allowed_extentions = []string{".html", ".css", ".js", ".jpg", ".jpeg", ".png", ".gif", ".swf", ".txt"}

// func main() {
// 	fmt.Println("test")
// }

func main() {
	i := 1
	DOCUMENT_ROOT := ""
	ncpu := 1
	var err error
	for i < len(os.Args) {
		switch os.Args[i] {
		case "-r":
			i += 1
			DOCUMENT_ROOT = os.Args[i]
		case "-c":
			i += 1
			ncpu, err = strconv.Atoi(os.Args[i])
			if err != nil {
				fmt.Println("-c is a numeric flag")
				os.Exit(-1)
			}
		default:
			i += 1
		}

	}
	runtime.GOMAXPROCS(ncpu)
	address, err := net.ResolveTCPAddr("tcp", "127.0.0.1"+":"+CONN_PORT)
	checkError(err)
	listener, err := net.ListenTCP(CONN_TYPE, address)
	for {
		conn, err := listener.Accept()
		if err == nil && conn != nil {
			go handleClient(conn, DOCUMENT_ROOT)
		} else {
			fmt.Println(err.Error())
		}
	}

}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

func handleClient(conn net.Conn, DOCUMENT_ROOT string) {
	defer conn.Close()
	var buf [1024 * 8]byte
	_, err := conn.Read(buf[0:])
	if err != nil {
		return
	}

	re11, _ := regexp.Compile(`(GET) (.*) HTTP.*`)
	re12, _ := regexp.Compile(`(HEAD) (.*) HTTP.*`)
	re2, _ := regexp.Compile(`(?m)^Host: (.*)`)

	header_type := re11.FindStringSubmatch(string(buf[:]))
	if header_type == nil { // if request is not GET
		header_type = re12.FindStringSubmatch(string(buf[:]))
		_ = re2.FindStringSubmatch(string(buf[:]))
	}
	if header_type != nil { // if request is GET or HEAD
		method := header_type[1]
		request := header_type[2]
		makeResponse(conn, request, method, DOCUMENT_ROOT)
	} else {
		response := fmt.Sprintf(STATUS, NOT_ALLOWED)
		_, _ = conn.Write([]byte(response))
		_, _ = conn.Write([]byte("\r\n"))
	}
}
func makeResponse(conn net.Conn, query, method, DOCUMENT_ROOT string) {
	url_path, _ := url.Parse(query)

	file_name, mime_type, err := determinate_mime(url_path.Path[1:]) // remove first slash
	STATUS_CODE := OK

	if err != nil {
		STATUS_CODE = NOT_FOUND
	}

	dat, local_code, err := check_n_read_file(DOCUMENT_ROOT, file_name)

	if err != nil {
		if local_code == "404" {
			STATUS_CODE = NOT_FOUND
		} else {
			STATUS_CODE = FORBIDDEN
		}
	}

	status := fmt.Sprintf(STATUS, STATUS_CODE)
	content_type := fmt.Sprintf(CONTENT_TYPE, mime_type)
	date := fmt.Sprintf(DATE, time.Now().Format(time.RFC850))
	content_length := fmt.Sprintf(CONTENT_LENGTH, len(dat))
	server := SERVER
	connection := CONNECTION

	_, _ = conn.Write([]byte(status))
	_, _ = conn.Write([]byte(date))
	_, _ = conn.Write([]byte(content_type))
	_, _ = conn.Write([]byte(content_length))
	_, _ = conn.Write([]byte(server))
	_, _ = conn.Write([]byte(connection))
	_, _ = conn.Write([]byte("\r\n"))

	if STATUS_CODE == OK && method == "GET" {
		_, _ = conn.Write(dat[0:])
	} else {
		return
	}
}
func get_mime_type_by_ext(extention string) string {
	result := ""
	switch extention {
	case ".html":
		result = "text/html"
	case ".txt":
		result = "text/plain"
	case ".jpg", ".jpeg":
		result = "image/jpeg"
	case ".png":
		result = "image/png"
	case ".gif":
		result = "image/gif"
	case ".css":
		result = "text/css"
	case ".js":
		result = "text/javascript"
	case ".swf":
		result = "application/x-shockwave-flash"
	default:
		result = ""
	}
	return result
}

func idx_after_ext(path string, allowed_extentions string) int {
	return strings.Index(path, allowed_extentions) + len(allowed_extentions)
}
func determinate_mime(file_name string) (string, string, error) {
	mime_type := get_mime_type_by_ext(".html")
	result_name := INDEX_FILE
	for s := range allowed_extentions {
		if strings.Contains(file_name, allowed_extentions[s]) {
			mime_type = get_mime_type_by_ext(allowed_extentions[s])
			sub_str_end_idx := idx_after_ext(file_name, allowed_extentions[s])
			result_name = file_name[0:sub_str_end_idx]
			return result_name, mime_type, nil
		}
	}
	if len(file_name) > 1 {
		result_name = file_name + INDEX_FILE
	}
	return result_name, mime_type, nil
}

func check_n_read_file(DOCUMENT_ROOT, file_name string) ([]byte, string, error) {
	code := "200"
	dat, err := ioutil.ReadFile(DOCUMENT_ROOT + file_name)
	if os.IsPermission(err) {
		code = "403"
	}
	if os.IsNotExist(err) {
		if strings.Contains(file_name, INDEX_FILE) {
			code = "403"
		} else {
			code = "404"
		}
	}
	return dat, code, err
}

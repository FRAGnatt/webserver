PATH := $(shell pwd)

all:		
	
		go build -o ./bin/httpd ./src/main.go
		chmod 777 bin/httpd



package main

import (
	"compress/flate"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

var (
	localport  = 8080
	remoteHost = ""
	isServer   = false
	listen     = ""
)

func HandleRequest(clientConn net.Conn) {
	timeout, _ := time.ParseDuration("1m")
	if hostConn, err := net.DialTimeout("tcp", remoteHost, timeout); err != nil {
		fmt.Println("err: dial " + remoteHost)
	} else {
		if !isServer {
			go func() { //host->client(decompress)
				zr := flate.NewReader(hostConn)
				io.Copy(clientConn, zr)
				hostConn.Close()
				clientConn.Close()
			}()
			go func() { //client->host(compress)
				zw, _ := flate.NewWriter(hostConn, flate.BestCompression)
				buf := make([]byte, 4096)
				for {
					n, err := clientConn.Read(buf)
					if n == 0 {
						if err == io.EOF {
							_ = clientConn.Close()
							break
						}
						time.Sleep(time.Millisecond * 10)
						continue
					}
					zw.Write(buf[:n])
					zw.Flush()
				}
				buf = nil
				zw.Close()
				hostConn.Close()
			}()
		} else {
			go func() { //host->client(compress)
				zw, _ := flate.NewWriter(clientConn, flate.BestCompression)
				buf := make([]byte, 4096)
				for {
					n, err := hostConn.Read(buf)
					if n == 0 {
						if err == io.EOF {
							_ = hostConn.Close()
							break
						}
						time.Sleep(time.Millisecond * 10)
						continue
					}
					zw.Write(buf[:n])
					zw.Flush()
				}
				buf = nil
				zw.Close()
				clientConn.Close()
			}()
			go func() { //client->host(decompress)
				zr := flate.NewReader(clientConn)
				io.Copy(hostConn, zr)
				clientConn.Close()
				hostConn.Close()
			}()
		}
	}
}

func main() {
	_localport := flag.Int("p", 8080, "local port")
	_remoteHost := flag.String("r", "", "remote host:port")
	_isServer := flag.Bool("s", false, "Server Mode")
	_listen := flag.String("l", "", "listen localhost")
	flag.Parse()
	localport = *_localport
	remoteHost = *_remoteHost
	isServer = *_isServer
	listen = *_listen

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", listen, localport))
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Println(fmt.Sprintf("Listening on %s:%d", listen, localport))
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		go HandleRequest(conn)
	}
}

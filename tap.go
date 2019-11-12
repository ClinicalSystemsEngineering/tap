package tap

import (
	"bufio"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

//take a pin and textmsg and return a correctly formatted TAP message
func createtapmsg(pin string, textmsg string) string {
	var sum int //the full checksum value
	var tapmsg string
	//tap messages have the format<STX>pin<CR>textmsg<CR><ETX><checksumchars><CR>
	//the following code will generate the checksumchars
	cr := rune(13)
	stx := rune(2)
	etx := rune(3)

	tapmsg = string(stx) + pin + string(cr) + textmsg + string(cr) + string(etx)

	//printabletapmsg := strings.Replace(strconv.QuoteToASCII(tapmsg), "\"", "", -1)
	//fmt.Printf("\n\nTapmsg:<%v>\n\n", printabletapmsg)
	sum = 0
	for _, runeval := range tapmsg {

		//	fmt.Printf("index:%v,runevalue:%v,quote:%v,value:%v\n", i, runeval, strconv.QuoteRune(runeval), int(runeval))

		sum += int(runeval)
	}
	d3 := 48 + sum - int(sum/16)*16
	sum = int(sum / 16)
	d2 := 48 + sum - int(sum/16)*16
	sum = int(sum / 16)
	d1 := 48 + sum - int(sum/16)*16
	d1rune := rune(d1)
	d2rune := rune(d2)
	d3rune := rune(d3)
	checksumchars := string(d1rune) + string(d2rune) + string(d3rune)
	//fmt.Printf("3valchecksum:%v\n", checksumchars)
	//fmt.Printf("d1:%v,d2:%v,d3:%v\n", d1, d2, d3)
	tapmsg += checksumchars + string(cr)
	return tapmsg
}

//Client starts a TAP client using server Adr
func Client(msgchan chan string, serverAdr string, verbose bool) {

	log.Printf("STARTING TAP Client to  %v...\n\n", serverAdr)
	tap, err := net.Dial("tcp", serverAdr)
	if err != nil {
		log.Printf("Error dialing TAP server: %v\n", err.Error())
		return
	}
	defer tap.Close()
	for {
		handler(tap, msgchan, verbose)
		tap.Close()
		tap, err = net.Dial("tcp", serverAdr)
		if err != nil {
			log.Printf("Error dialing TAP server: %v\n", err.Error())
			return
		}
	}

}
func initTap(c net.Conn, verbose bool) bool {

	timeoutDuration := 5 * time.Second

	r := bufio.NewReader(c)

	//cr := rune(13)
	esc := rune(27)
	//ack := rune(6)
	//nak := rune(21)
	//nullrune := rune(0)

	bytes := make([]byte, 100)

RetryInit:
	if verbose {
		log.Println("Writing <CR> to TAP connection")
	}

	c.SetWriteDeadline(time.Now().Add(timeoutDuration))
	_, err := c.Write([]byte("\r"))
	if err != nil {
		log.Printf("Error writing <CR> on TAP connection: %v\n", err.Error())
		log.Println("Closing connection and awaiting new connection.")
		c.Close()
		return false
	}
	c.SetReadDeadline(time.Now().Add(timeoutDuration))
	num, err := r.Read(bytes)
	if err != nil {

		log.Printf("Error reading response ID= from TAP connection: %v\n", err.Error())
		log.Println("Closing connection and awaiting new connection.")
		c.Close()
		return false
	}
	response := string(bytes[0:3])

	if verbose {
		log.Printf("TAP response:%v\n", strconv.QuoteToASCII(response))
	} //response should be ID=

	if response != "ID=" {
		goto RetryInit
	}

	if verbose {
		log.Println("Writing <ESC>PG1<CR> to TAP connection")
	}
	c.SetWriteDeadline(time.Now().Add(timeoutDuration))
	_, err = c.Write([]byte(string(esc) + "PG1\r"))
	if err != nil {
		log.Printf("Error writing <ESC>PG1<CR> to TAP connection: %v\n", err.Error())
		log.Println("Closing connection and awaiting new connection.")
		c.Close()
		return false
	}
	c.SetReadDeadline(time.Now().Add(timeoutDuration))
	num, err = r.Read(bytes)
	if err != nil {

		log.Printf("Error reading leading optional message response from TAP client: %v\n", err.Error())
		log.Println("Closing connection and awaiting new connection.")
		c.Close()
		return false
	}
	response = string(bytes[0:num])
	if verbose {
		log.Printf("TAP response:%v\n", strconv.QuoteToASCII(response))
	} //response should be optional coded message<CR>

	c.SetReadDeadline(time.Now().Add(timeoutDuration))
	num, err = r.Read(bytes)
	if err != nil {

		log.Printf("Error reading PG1 ack/nak response from TAP client: %v\n", err.Error())
		log.Println("Closing connection and awaiting new connection.")
		c.Close()
		return false
	}
	response = string(bytes[0:num])
	if verbose {
		log.Printf("TAP response:%v\n", strconv.QuoteToASCII(response))
	} //response should be optional coded message<CR>
	if strings.Contains(response, "[p") {
		goto EndInit
	}

	c.SetReadDeadline(time.Now().Add(timeoutDuration))
	num, err = r.Read(bytes)
	if err != nil {

		log.Printf("Error reading optional or [p response from TAP connection: %v\n", err.Error())
		log.Println("Closing connection and awaiting new connection.")
		c.Close()
		return false
	}
	response = string(bytes[0:num])
	if verbose {
		log.Printf("TAP response:%v\n", strconv.QuoteToASCII(response))
	} //response should be optional coded message<CR>

	if strings.Contains(response, "[p") {
		goto EndInit
	}

	c.SetReadDeadline(time.Now().Add(timeoutDuration))
	num, err = r.Read(bytes)
	if err != nil {

		log.Printf("Error reading [p message response from TAP client: %v\n", err.Error())
		log.Println("Closing connection and awaiting new connection.")
		c.Close()
		return false
	}
	response = string(bytes[0:num])
	if verbose {
		log.Printf("TAP response:%v\n", strconv.QuoteToASCII(response))
	} //response should be optional coded message<CR>

	if strings.Contains(response, "[p") {
		goto EndInit
	}
EndInit:

	log.Print("TAP connection initialized\n\n")

	return true

}

func handler(c net.Conn, parsedmsgsqueue chan string, verbose bool) {

	timeoutDuration := 5 * time.Second
	initTimer := time.NewTimer(timeoutDuration)

	r := bufio.NewReader(c)

	init := false
	//cr := rune(13)
	//esc := rune(27)
	//ack := rune(6)
	nak := rune(21)
	//nullrune := rune(0)
	eot := rune(4)

	bytes := make([]byte, 100)
	var num int

	for {

		select {
		case msg, ok := <-parsedmsgsqueue:
			if ok {

				if init == false {
					init = initTap(c, verbose) //initialize tap server to receive messages
					if init == false {         //there was an issue with init
						log.Printf("Error initializing TAP. Placing %v back on the TAP queue.\n", msg)
						parsedmsgsqueue <- msg
						log.Println("Returning from connection handler and awaiting new connection.")
						return
					}
				}
				splitmsg := strings.Split(msg, ";")
				pin, text := splitmsg[0], splitmsg[1]
				//fmt.Printf("received pin:%v\nrecieved textmsg:%v\n", pin, textmsg)
				tapmsg := createtapmsg(pin, text)
			RetryMsg:

				c.SetWriteDeadline(time.Now().Add(timeoutDuration))
				_, err := c.Write([]byte(tapmsg))
				if err != nil {
					log.Printf("Error writing msg to TAP connection: %v\n", err.Error())
					log.Printf("Placing %v back on the TAP queue.\n", msg)
					parsedmsgsqueue <- msg
					log.Println("Closing connection and awaiting new connection.")
					c.Close()
					return
				}

				log.Printf("Sent <%v> to TAP client\n", strconv.QuoteToASCII(tapmsg))
				//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response)) //should be coded response
				c.SetReadDeadline(time.Now().Add(timeoutDuration))
				num, err = r.Read(bytes)
				if err != nil {
					log.Printf("Error reading ack/nak response from TAP client: %v\n", err.Error())
					log.Printf("Placing %v back on the TAP queue.\n", msg)
					parsedmsgsqueue <- msg
					log.Println("Closing connection and awaiting new connection.")
					c.Close()
					return
				}

				response := string(bytes[0:num])
				//fmt.Printf("\n\nTAP response:%v\n\n", strings.Replace(strconv.QuoteToASCII(response), "\"", "", -1)) //should be ack/nak
				if strings.Contains(response, string(nak)) {

					time.Sleep(1 * time.Second)
					goto RetryMsg
				}
				initTimer = time.NewTimer(timeoutDuration) //reset idle timer
			}
		case <-initTimer.C:
			if init == true { //close transmission window and wait for messages to land on queue
				init = false
				if verbose {
					log.Println("Sending End of Transmission")
				}

				c.SetWriteDeadline(time.Now().Add(timeoutDuration))
				_, err := c.Write([]byte(string(eot) + "\r"))
				if err != nil {
					log.Printf("Error EOT writing msg to TAP connection: %v\n", err.Error())
					log.Println("Closing connection and awaiting new connection.")
					c.Close()
					return
				}

				c.SetReadDeadline(time.Now().Add(timeoutDuration))
				num, err = r.Read(bytes)
				if err != nil {
					log.Printf("Error reading EOT message response from TAP client: %v\n", err.Error())
					log.Println("Closing connection and awaiting new connection.")
					c.Close()
					return
				}

				response := string(bytes[0:num])
				if verbose {
					log.Printf("TAP response:%v\n", strconv.QuoteToASCII(response))
				} //response should be optional coded message<CR>

				c.SetReadDeadline(time.Now().Add(timeoutDuration))
				num, err = r.Read(bytes)
				if err != nil {
					log.Printf("Error reading EOT response from TAP client: %v\n", err.Error())
					log.Println("Closing connection and awaiting new connection.")
					c.Close()
					return
				}

				response = string(bytes[0:num])
				if verbose {
					log.Printf("TAP response:%v\n", strconv.QuoteToASCII(response))
				} //response should be <ESC><EOT><CR>
				if verbose {
					log.Println("No message to process on queue waiting for a message...")
				}
				initTimer = time.NewTimer(timeoutDuration) //reset idle time
			} else { //wait for message to land on queue
				//log.Println("No message to process on queue waiting for a message...")
				initTimer = time.NewTimer(timeoutDuration) //reset idle timer
			}

		}
	}

}

//Server starts a Tap server using portnum as the port or 10001 if not specified
func Server(msgchan chan string, portnum string, whitelist string, verbose bool) {
	log.Printf("STARTING TAP listener on tcp port %v...\n\n", portnum)
	log.Printf("TAP Whitelisted: %v\n", whitelist)
	tap, err := net.Listen("tcp", ":"+portnum)
	if err != nil {
		//	fmt.Println("Error opening tap output, check log for details")
		log.Fatal(err)
	}
	defer tap.Close()

	//accept connetions and start a TAP server handler against them
	//==========================================================================
	for {
		tapconn, err := tap.Accept()
		if err != nil {

			log.Printf("Error accepting a TAP connection: %v\n", err.Error())
			log.Println("Closing connection and awaiting new connection.")
			tapconn.Close()

		}
		//check if incoming connection is on the tap white list
		addr, ok := tapconn.RemoteAddr().(*net.TCPAddr)
		if !ok {
			log.Fatal("Error reading incoming TAP connection ip address")
		}

		log.Printf("Received TAP connection request from %v\n", addr.IP.String())
		// if not in the whitelist close the connecition
		if addr.IP.String() != whitelist && whitelist != "127.0.0.1" {
			log.Printf("Client ip %v not on whitelist. Closing connection.\n", addr.IP.String())
			tapconn.Close()
		} else //if on  the whitelist handle the connection
		{
			log.Printf("TAP Client ip %v accepted. Handling connection.\n", addr.IP.String())
			go handler(tapconn, msgchan, verbose)
		}
	}
	//==========================================================================
}

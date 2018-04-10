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
func Client(msgchan chan string, serverAdr string) {

	log.Printf("STARTING TAP Client to  %v...\n\n", serverAdr)
	tap, err := net.Dial("tcp", serverAdr)
	if err != nil {
		log.Printf("Error dialing TAP server: %v\n", err.Error())
		return
	}
	defer tap.Close()
	for {
		handler(tap, msgchan)
		tap.Close()
		tap, err = net.Dial("tcp", serverAdr)
		if err != nil {
			log.Printf("Error dialing TAP server: %v\n", err.Error())
			return
		}
	}

}

func handler(c net.Conn, parsedmsgsqueue chan string) {

	timeoutDuration := 5 * time.Second

	r := bufio.NewReader(c)

	init := true
	//cr := rune(13)
	esc := rune(27)
	//ack := rune(6)
	nak := rune(21)

	if init == true { //initialize the tap server

	RetryInit:

		c.SetWriteDeadline(time.Now().Add(timeoutDuration))
		_, err := c.Write([]byte("BYE\r\r"))
		if err != nil {
			log.Printf("Error writing BYE from TAP client: %v\n", err.Error())
			log.Println("Closing connection and awaiting new connection.")
			c.Close()
			return
		}
		c.SetReadDeadline(time.Now().Add(timeoutDuration))
		response, err := r.ReadString('\r')
		if err != nil {

			log.Printf("Error reading response from TAP client: %v\n", err.Error())
			log.Println("Closing connection and awaiting new connection.")
			c.Close()
			return
		}
		//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response))
		if response != "ID=\r" {
			goto RetryInit
		}

		c.SetWriteDeadline(time.Now().Add(timeoutDuration))
		_, err = c.Write([]byte(string(esc) + "PG1\r"))
		if err != nil {
			log.Printf("Error writing PG1 from TAP client: %v\n", err.Error())
			log.Println("Closing connection and awaiting new connection.")
			c.Close()
			return
		}
		c.SetReadDeadline(time.Now().Add(timeoutDuration))
		response, err = r.ReadString('\r')
		if err != nil {

			log.Printf("Error reading response from TAP client: %v\n", err.Error())
			log.Println("Closing connection and awaiting new connection.")
			c.Close()
			return
		}
		//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response))
		c.SetReadDeadline(time.Now().Add(timeoutDuration))
		response, err = r.ReadString('\r')
		if err != nil {

			log.Printf("Error reading response from TAP client: %v\n", err.Error())
			log.Println("Closing connection and awaiting new connection.")
			c.Close()
			return
		}
		//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response)) //response should be 110 1.8
		c.SetReadDeadline(time.Now().Add(timeoutDuration))
		response, err = r.ReadString('\r')
		if err != nil {

			log.Printf("Error reading response from TAP client: %v\n", err.Error())
			log.Println("Closing connection and awaiting new connection.")
			c.Close()
			return
		}
		//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response)) //response should be optional message
		c.SetReadDeadline(time.Now().Add(timeoutDuration))
		response, err = r.ReadString('\r')
		if err != nil {

			log.Printf("Error reading response from TAP client: %v\n", err.Error())
			log.Println("Closing connection and awaiting new connection.")
			c.Close()
			return
		}
		//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response)) //response should be <ESC>[p<CR> otherwise retry keep alive
		if !strings.Contains(response, "[p") {
			goto RetryInit
		}
		init = false
		//fmt.Print("\n\nTAP server initialized\n\n")

	}
	//time.Sleep(10 * time.Second)

	for {

		select {
		case msg, ok := <-parsedmsgsqueue:
			if ok {
				splitmsg := strings.Split(msg, ";")
				pin, text := splitmsg[0], splitmsg[1]
				//fmt.Printf("received pin:%v\nrecieved textmsg:%v\n", pin, textmsg)
				tapmsg := createtapmsg(pin, text)
			RetryMsg:

				c.SetWriteDeadline(time.Now().Add(timeoutDuration))
				_, err := c.Write([]byte(tapmsg))
				if err != nil {
					log.Printf("Error writing PG1 from TAP client: %v\n", err.Error())
					log.Println("Closing connection and awaiting new connection.")
					parsedmsgsqueue <- msg
					c.Close()
					return
				}

				c.SetReadDeadline(time.Now().Add(timeoutDuration))
				response, err := r.ReadString('\r')
				if err != nil {

					log.Printf("Error reading response from TAP client: %v\n", err.Error())
					log.Printf("Placing %v back on the TAP queue.\n", msg)
					parsedmsgsqueue <- msg
					log.Println("Closing connection and awaiting new connection.")
					c.Close()
					return

				}
				log.Printf("Sent <%v> to TAP client\n", strconv.QuoteToASCII(tapmsg))
				//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response)) //should be coded response
				c.SetReadDeadline(time.Now().Add(timeoutDuration))
				response, err = r.ReadString('\r')
				if err != nil {
					log.Printf("Error reading response from TAP client: %v\n", err.Error())
					log.Println("Closing connection and awaiting new connection.")
					c.Close()
					return
				}
				//fmt.Printf("\n\nTAP response:%v\n\n", strings.Replace(strconv.QuoteToASCII(response), "\"", "", -1)) //should be ack/nak
				if strings.Contains(response, string(nak)) {

					time.Sleep(1 * time.Second)
					goto RetryMsg
				}
			}
		default:
			//do a keep alive sequence
			// RetryKA:
			// 	//fmt.Printf("\n\nNo tap value ready, moving on with keep alive.\n\n")

			// 	c.SetWriteDeadline(time.Now().Add(timeoutDuration))
			// 	_, err = c.Write([]byte("\r"))
			// 	if err != nil {
			// 		log.Printf("Error writing <CR> from TAP client: %v\n", err.Error())
			// 		log.Println("Closing connection and awaiting new connection.")
			// 		c.Close()
			// 		return
			// 	}
			// 	c.SetReadDeadline(time.Now().Add(timeoutDuration))
			// 	response, err := r.ReadString('\r')
			// 	if err != nil {
			// 		log.Printf("Error reading response from TAP client: %v\n", err.Error())
			// 		log.Println("Closing connection and awaiting new connection.")
			// 		c.Close()
			// 		return
			// 	}
			// 	//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response))
			// 	if response != "ID=\r" {
			// 		goto RetryKA
			// 	}

			// 	c.SetWriteDeadline(time.Now().Add(timeoutDuration))
			// 	_, err = c.Write([]byte(string(esc) + "PG1\r"))
			// 	if err != nil {
			// 		log.Printf("Error writing PG1 from TAP client: %v\n", err.Error())
			// 		log.Println("Closing connection and awaiting new connection.")
			// 		c.Close()
			// 		return
			// 	}
			// 	c.SetReadDeadline(time.Now().Add(timeoutDuration))
			// 	response, err = r.ReadString('\r')
			// 	if err != nil {
			// 		log.Printf("Error reading response from TAP client: %v\n", err.Error())
			// 		log.Println("Closing connection and awaiting new connection.")
			// 		c.Close()
			// 		return
			// 	}
			// 	c.SetReadDeadline(time.Now().Add(timeoutDuration))
			// 	response, err = r.ReadString('\r')
			// 	if err != nil {
			// 		log.Printf("Error reading response from TAP client: %v\n", err.Error())
			// 		log.Println("Closing connection and awaiting new connection.")
			// 		c.Close()
			// 		return
			// 	}
			// 	//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response)) //response should be 110 1.8
			// 	c.SetReadDeadline(time.Now().Add(timeoutDuration))
			// 	response, err = r.ReadString('\r')
			// 	if err != nil {
			// 		log.Printf("Error reading response from TAP client: %v\n", err.Error())
			// 		log.Println("Closing connection and awaiting new connection.")
			// 		c.Close()
			// 		return
			// 	}
			// 	//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response)) //response should be optional message
			// 	c.SetReadDeadline(time.Now().Add(timeoutDuration))
			// 	response, err = r.ReadString('\r')
			// 	if err != nil {
			// 		log.Printf("Error reading response from TAP client: %v\n", err.Error())
			// 		log.Println("Closing connection and awaiting new connection.")
			// 		c.Close()
			// 		return
			// 	}
			// 	//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response)) //response should be <ESC>[p<CR> otherwise retry keep alive
			// 	if !strings.Contains(response, "[p") {
			// 		//fmt.Print("\n\n Error in TAP keep alive did not find [p ready state retrying keep alive \n\n")
			// 		goto RetryKA
			// 	}
			log.Println("No message to process on queue sleeping 5 sec...")
			time.Sleep(5 * time.Second)
			//ready for next submit
		}
	}

}

//Server starts a Tap server using portnum as the port or 10001 if not specified
func Server(msgchan chan string, portnum string) {
	log.Printf("STARTING TAP listener on tcp port %v...\n\n", portnum)

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
		go handler(tapconn, msgchan)
	}
	//==========================================================================
}

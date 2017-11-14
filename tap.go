package tap

import (
	"bufio"
	"log"
	"net"
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

//Server starts a Tap server using portnum as the port or 10001 if not specified
func Server(msgchan <-chan string, portnum string) {
	log.Printf("STARTING TAP listener on tcp port %v...\n\n", portnum)
	tap, err := net.Listen("tcp", ":"+portnum)
		if err != nil {
			//	fmt.Println("Error opening tap output, check log for details")
			log.Fatal(err)
		}
	defer tap.Close()
	
	//==========================================================================
	for {
		tapconn, err := tap.Accept()
		if err != nil {
			//	fmt.Println("Error accepting a TAP connection, check log for details")
			tapconn.Close()
			log.Print(err.Error())

		}
		go func(c net.Conn, parsedmsgsqueue <-chan string) {
			//	fmt.Print("\n\nAccepted TAP connection Started TAP output routine..\n\n")
			r := bufio.NewReader(c)

			init := true
			//cr := rune(13)
			esc := rune(27)
			//ack := rune(6)
			nak := rune(21)

			if init == true { //initialize the tap server
				//fmt.Print("\n\nStarting TAP init.\n\n")
			RetryInit:

				c.Write([]byte("BYE\r\r"))
				//fmt.Printf("\n\nTried to write to TAP output<CR>\n\n")
				response, err := r.ReadString('\r')
				if err != nil {
					//fmt.Printf("\n\nerror reading response from tap server\n\n")
					log.Print(err.Error())
					c.Close()
					return
				}
				//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response))
				if response != "ID=\r" {
					goto RetryInit
				}

				c.Write([]byte(string(esc) + "PG1\r"))
				response, err = r.ReadString('\r')
				if err != nil {
					//fmt.Printf("\n\nerror reading response from tap server\n\n")
					log.Print(err.Error())
					c.Close()
					return
				}
				//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response))
				response, err = r.ReadString('\r')
				if err != nil {
					//fmt.Printf("\n\nerror reading response from tap server\n\n")
					log.Print(err.Error())
					c.Close()
					return
				}
				//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response)) //response should be 110 1.8
				response, err = r.ReadString('\r')
				if err != nil {
					//fmt.Printf("\n\nerror reading response from tap server\n\n")
					log.Print(err.Error())
					c.Close()
					return
				}
				//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response)) //response should be optional message
				response, err = r.ReadString('\r')
				if err != nil {
					//fmt.Printf("\n\nerror reading response from tap server\n\n")
					log.Print(err.Error())
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

						c.Write([]byte(tapmsg))
						response, err := r.ReadString('\r')
						if err != nil {
							//fmt.Printf("\n\nerror reading response from tap server\n\n")
							log.Print(err.Error())
							c.Close()
							return

						}
						//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response)) //should be coded response

						response, err = r.ReadString('\r')
						if err != nil {
							//fmt.Printf("\n\nerror reading response from tap server")
							log.Print(err.Error())
							c.Close()
							return
						}
						//fmt.Printf("\n\nTAP response:%v\n\n", strings.Replace(strconv.QuoteToASCII(response), "\"", "", -1)) //should be ack/nak
						if strings.Contains(response, string(nak)) {

							time.Sleep(1 * time.Second)
							goto RetryMsg
						}
					}
				default: //do a keep alive sequence
				RetryKA:
					//fmt.Printf("\n\nNo tap value ready, moving on with keep alive.\n\n")

					c.Write([]byte("\r"))
					response, err := r.ReadString('\r')
					if err != nil {
						//fmt.Printf("\n\nerror reading response from tap server")
						log.Print(err.Error())
						c.Close()
						return
					}
					//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response))
					if response != "ID=\r" {
						goto RetryKA
					}

					c.Write([]byte(string(esc) + "PG1\r"))
					response, err = r.ReadString('\r')
					if err != nil {
						//fmt.Printf("\n\nerror reading response from tap server\n\n")
						log.Print(err.Error())
						c.Close()
						return
					}
					response, err = r.ReadString('\r')
					if err != nil {
						//fmt.Printf("\n\nerror reading response from tap server\n\n")
						log.Print(err.Error())
						c.Close()
						return
					}
					//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response)) //response should be 110 1.8
					response, err = r.ReadString('\r')
					if err != nil {
						//fmt.Printf("\n\nerror reading response from tap server\n\n")
						log.Print(err.Error())
						c.Close()
						return
					}
					//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response)) //response should be optional message
					response, err = r.ReadString('\r')
					if err != nil {
						//fmt.Printf("\n\nerror reading response from tap server\n\n")
						log.Print(err.Error())
						c.Close()
						return
					}
					//fmt.Printf("\n\nTAP response:%v\n\n", strconv.QuoteToASCII(response)) //response should be <ESC>[p<CR> otherwise retry keep alive
					if !strings.Contains(response, "[p") {
						//fmt.Print("\n\n Error in TAP keep alive did not find [p ready state retrying keep alive \n\n")
						goto RetryKA
					}
					time.Sleep(5 * time.Second)
					//ready for next submit
				}
			}

		}(tapconn, msgchan)
		//==========================================================================

	}
}

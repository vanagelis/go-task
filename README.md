<b>Sending messages:</b><br>
Messages can be sent by performing HTTP POST request to /infocenter/{topic}
route. Topic for	the	message is	passed	in	the request	URL,	in	place of	{topic} tag.
For	example	client	that	wants	to	send	message	"labas"	to	a	topic	"test",	should	perform
the following	request:<br>
POST /infocenter/test HTTP/1.0<br>
Host: localhost<br>
Content-length: 5<br>
labas<br>
HTTP/1.0 204 No Content<br>
Date: Mon, 14 Sep 2015 08:26:20 GMT<br>
Server response	should	be HTTP	204	code if message	was accepted	successfully.

<b>Receiving messages:</b><br>
Messages	can	be	received	by	subscripting	to	a	given	topic,	which	can	be	done	by	sending
HTTP GET	request	to the	same API	route /infocenter/{topic}.
Response	 to	 this	 request	 should	 be	 an	 event	 stream	 (as	 defined	 in	W3C	 specification
"Server-Sent	Events"	http://www.w3.org/TR/eventsource/).	All	 sent	messages	 should
have a	message	type	msg.
Service	should	disconnect	all	clients	 that	are	consuming	 the	stream	 for	more	 than	 the
max allowed time (for example 30 sec). Before client is disconnected, server should
send a special timeout event. The contents of the timeout event should be the time how
long client	was	connected.	Note	that	this	event	is	not	a	message.

Example of receiving message events:
GET /infocenter/test HTTP/1.0
Host: localhost
HTTP/1.0 200 OK
Cache-Control: no-cache
Content-Type: text/event-stream
Date: Mon, 14 Sep 2015 08:33:46 GMT
id: 7
event: msg
data: labas
event: timeout
data: 30s

#!/usr/bin/python
import re, time, SocketServer
from pixarray import PixArray

class MyTCPHandler(SocketServer.StreamRequestHandler):
  def _parse_color(self, color_str):
    r, g, b = color_str[:2], color_str[2:4], color_str[4:]
    r, g, b = [int(n, 16) for n in (r, g, b)]
    return (r, g, b)
    
  def _parse_time(self, time_str, default):
    if time_str==None:
      return default
    return float(time_str)

  def handle(self):
    while True:
      line = self.rfile.readline()
      if line=="":
        break
      line=line.strip()
      if line=="QUIT":
        print "QUIT"
        self.wfile.write("OK\n")
        break
      m=re.match('^FADE_ALL ([a-zA-Z0-9]{6})( [0-9]+\.[0-9]+)?$',line)
      if m!=None:
	print "FADE_ALL %s %s" % (m.group(1), m.group(2))
	r, g, b = self._parse_color(m.group(1))
	t = self._parse_time(m.group(2), 1.0)
	self.pa.fade_all(r, g, b, t)
	self.wfile.write("OK\n")
	continue
      m=re.match('^SET_ALL ([a-zA-Z0-9]{6})$',line)
      if m!=None:
	print "SET_ALL %s" % (m.group(1))
	r, g, b = self._parse_color(m.group(1))
	self.pa.set_all(r, g, b)
	self.wfile.write("OK\n")
      m=re.match('^ZIP_SET_ALL ([a-zA-Z0-9]{6})( [0-9]+\.[0-9]+)?( [UD])?$',line)
      if m!=None:
	print "ZIP_SET_ALL %s %s %s" % (m.group(1), m.group(2), m.group(3))
	r, g, b = self._parse_color(m.group(1))
	t = self._parse_time(m.group(2), 1.0)
	if m.group(3)==" D":
	  d = -1
        else:
          d = 1
	print "ZIP_SET_ALL %d/%d/%d %f %d" % (r,g,b,t,d)
	self.pa.zip_set_all(r, g, b, t, d)
	self.wfile.write("OK\n")

dev = "/dev/spidev0.0"
spidev = file(dev, "wb")

pa = PixArray(1*32, spidev)
pa.set_all(0,0,0)

SocketServer.TCPServer.allow_reuse_address = True
MyTCPHandler.pa=pa
server = SocketServer.TCPServer(("0.0.0.0", 24601), MyTCPHandler)
server.serve_forever()

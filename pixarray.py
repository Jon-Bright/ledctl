import RPi.GPIO as GPIO, time

class PixArray:
  def __init__(self, num_pixels, spi_dev):
    self.num_reset=(num_pixels+31)/32
    self.num_pixels=num_pixels
    self.spi_dev=spi_dev
    self.pix_val=bytearray((self.num_pixels * 3) + self.num_reset)
    for x in range(num_pixels):
      self.pix_val[x]=0x80
    firstreset = bytearray(self.num_reset)
    self.spi_dev.write(firstreset)
    self.spi_dev.flush()
  
  def _write(self):
    self.spi_dev.write(self.pix_val)
    self.spi_dev.flush()

  def set_all(self, r, g, b):
    for x in range(self.num_pixels):
      self.pix_val[x*3] = 0x80 | g
      self.pix_val[x*3+1] = 0x80 | r
      self.pix_val[x*3+2] = 0x80 | b
    self._write()

  def set_one(self, ix, r, g, b):
    self.pix_val[ix*3] = 0x80 | g
    self.pix_val[ix*3+1] = 0x80 | r
    self.pix_val[ix*3+1] = 0x80 | b
    self._write()

  def zip_set_all(self, r, g, b, t, d):
    ts = time.time()
    te = ts+t
    if d<0:
      already_set=self.num_pixels-1
    else:
      already_set=0
    while True:
      tn = time.time()
      if (tn>=te):
        self.set_all(r,g,b)
        return
      td=tn-ts
      pix_set=int(self.num_pixels*(td/t))
      if d<0:
        pix_set=self.num_pixels-1-pix_set
      for x in range(already_set,pix_set,d):
        self.pix_val[x*3] = 0x80 | g
        self.pix_val[x*3+1] = 0x80 | r
        self.pix_val[x*3+2] = 0x80 | b
        self._write()
      already_set=pix_set

  def knight_rider(self, t):
    ts = time.time()
    te = ts+t
    self.set_all(1,0,0)
    pulse_len=self.num_pixels/4
    pulse_head=0
    pulse_time=1.5
    while True:
      tn = time.time()
      if (tn>=te):
        self.set_all(0,0,0)
        return
      td=tn-ts
      pulse_num=int(td/pulse_time)
      if (pulse_num%2)==0:
        pulse_dir=1
      else:
        pulse_dir=-1
      pulse_prog=(td-(pulse_num*pulse_time))/pulse_time
      pulse_head=int((self.num_pixels+pulse_len)*pulse_prog)
      if pulse_dir==-1:
        pulse_head=self.num_pixels-pulse_head
      pulse_tail=pulse_head+(pulse_dir*pulse_len*-1)
      if pulse_tail<0:
        pulse_tail=0
      elif pulse_tail>=self.num_pixels:
        pulse_tail=self.num_pixels-1
      if pulse_head<0:
        range_head=0
      elif pulse_head>=self.num_pixels:
        range_head=self.num_pixels-1
      else:
        range_head=pulse_head
      for x in range(pulse_tail, range_head, pulse_dir):
        v=int(((pulse_len-abs(pulse_head-x))/float(pulse_len))*126.0)+1
        self.pix_val[x*3+1] = 0x80 | v
      self._write()

  def fade_all(self, r, g, b, t):
    ts = time.time()
    te = ts+t
    startvals = bytearray(self.num_pixels*3)
    startvals[:] = self.pix_val
    for x in range(self.num_pixels*3):
      startvals[x] = startvals[x] & 0x7f
    diffs = []
    for x in range(self.num_pixels):
      diffs.append(g-startvals[x*3])
      diffs.append(r-startvals[x*3+1])
      diffs.append(b-startvals[x*3+2])
    while True:
      tn = time.time()
      if (tn>=te):
        self.set_all(r, g, b)
        return
      p=(tn-ts)/t
      for x in range(self.num_pixels):
        self.pix_val[x*3] = 0x80 | int(startvals[x*3]+(diffs[x*3]*p))
        self.pix_val[x*3+1] = 0x80 | int(startvals[x*3+1]+(diffs[x*3+1]*p))
        self.pix_val[x*3+2] = 0x80 | int(startvals[x*3+2]+(diffs[x*3+2]*p))
      self._write()

  def fade_all_to_ranges(self, ranges, t):
    ts = time.time()
    te = ts+t
    startvals = bytearray(self.num_pixels*3)
    startvals[:] = self.pix_val
    for x in range(self.num_pixels*3):
      startvals[x] = startvals[x] & 0x7f
    endvals = bytearray(self.num_pixels*3)
    num_ranges=len(ranges)-1
    if (num_ranges<1):
      raise ValueError('Must have at least one range')
    range_ix=0
    for x in range(self.num_pixels):
      while True:
        if (len(ranges[range_ix+1])==4):
          my_end=ranges[range_ix+1][3]
        else:
          my_end=(float(self.num_pixels)/num_ranges)*(range_ix+1)
        if x<int(my_end):
          break
        range_ix=range_ix+1
      if (len(ranges[range_ix])==4):
        my_start=ranges[range_ix][3]
      else:
        my_start=(float(self.num_pixels)/num_ranges)*range_ix
      my_len=my_end-my_start
      my_pos=(float(x)-my_start)/my_len
      amt_ra=1.0-my_pos
      amt_rb=my_pos
      endvals[x*3]=int(float(ranges[range_ix][1])*amt_ra+float(ranges[range_ix+1][1])*amt_rb)
      endvals[x*3+1]=int(float(ranges[range_ix][0])*amt_ra+float(ranges[range_ix+1][0])*amt_rb)
      endvals[x*3+2]=int(float(ranges[range_ix][2])*amt_ra+float(ranges[range_ix+1][2])*amt_rb)
    diffs = []
    for x in range(self.num_pixels*3):
      diffs.append(endvals[x]-startvals[x])
    while True:
      tn = time.time()
      if (tn>=te):
        for x in range(self.num_pixels*3):
          self.pix_val[x] = 0x80 | endvals[x]
        return
      p=(tn-ts)/t
      for x in range(self.num_pixels):
        self.pix_val[x*3] = 0x80 | int(startvals[x*3]+(diffs[x*3]*p))
        self.pix_val[x*3+1] = 0x80 | int(startvals[x*3+1]+(diffs[x*3+1]*p))
        self.pix_val[x*3+2] = 0x80 | int(startvals[x*3+2]+(diffs[x*3+2]*p))
      self._write()

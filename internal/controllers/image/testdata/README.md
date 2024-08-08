These test artifacts are used by the upload test.

* raw.img  
  1MB of zeroes  
  `dd if=/dev/zero of=raw.img bs=1M count=1`

* raw.img.bz  
  raw.img compressed with bzip2  
  `bzip2 < raw.img > raw.img.bz2`

* raw.img.gz  
  raw.img compressed with gzip  
  `gzip < raw.img > raw.img.gz`

* raw.img.xz  
  raw.img compressed with xz  
  `xz < raw.img > raw.img.gz`

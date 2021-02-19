# Merry
file download server with dynamic bandwidth throttling


## Feature
#### 1.danymic bandwidth throttling
#### 2.quic protocol (udp based)
#### 3.resume from break-point


## Build
./build.sh


## Usage
#### server
```
./ser
```
#### client
```
1.set bandwidth
./cli -s 127.0.0.1:8282 -b 1024

2.download file
./cli -s 127.0.0.1:8282 -f /tmp/a.txt -o /tmp/b.txt
```

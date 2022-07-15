# Pakcet Buffer BenchMark

Tried the following ways to buffer the packets
- previous circular buffer
- slice
- golang chan to buffer packet
- linked list from "container/list"

Here are the results from the Macbook:

## circular buffer

![img.png](docs/img.png)

## slice

![img_1.png](docs/img_1.png)

## chan

![img_2.png](docs/img_2.png)

## container/list

![img_3.png](docs/img_3.png)
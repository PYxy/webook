package test_demo

import (
	//"fmt"
	//"log"
	//"net/http"
	//"testing"
	//
	//"github.com/google/gopacket"
	//"github.com/google/gopacket/layers"
	//"github.com/google/gopacket/pcap"
	"fmt"
	"log"
	"net/http"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

func Test_tt(t *testing.T) {
	//packetData := []byte{ /* 你的数据包字节流 */ }
	//
	//// 创建一个Packet对象
	//packet := gopacket.NewPacket(packetData, layers.LayerTypeEthernet, gopacket.Default)
	//
	//// 遍历数据包的所有层
	//for _, layer := range packet.Layers() {
	//	// 检查每一层的类型是否为SSH协议层
	//	if layer.LayerType() == layers.SSH {
	//		fmt.Println("SSH协议层存在")
	//	}
	//}
	// 打开网络接口或者读取pcap文件
	handle, err := pcap.OpenLive("eth0", 65536, true, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	// 设置过滤器，仅捕获HTTP流量
	err = handle.SetBPFFilter("tcp and port 80")
	if err != nil {
		log.Fatal(err)
	}

	// 创建数据包源
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	// 迭代处理每个数据包
	for packet := range packetSource.Packets() {
		// 解析Ethernet层
		ethLayer := packet.Layer(layers.LayerTypeEthernet)
		if ethLayer == nil {
			continue
		}
		ethernetPacket, _ := ethLayer.(*layers.Ethernet)

		// 解析IPv4层
		ipLayer := packet.Layer(layers.LayerTypeIPv4)
		if ipLayer == nil {
			continue
		}
		ipPacket, _ := ipLayer.(*layers.IPv4)

		// 解析TCP层
		tcpLayer := packet.Layer(layers.LayerTypeTCP)
		if tcpLayer == nil {
			continue
		}
		tcpPacket, _ := tcpLayer.(*layers.TCP)

		// 解析HTTP层
		appLayer := packet.ApplicationLayer()
		appLayer.LayerType() == layers.
		if appLayer != nil {
			// 使用gopacket的http库解析HTTP层数据
			httpPacket, err := http.ReadRequest(appLayer.Payload())
			if err == nil {
				// 获取HTTP URL
				url := httpPacket.URL.String(h)
				fmt.Println("HTTP URL:", url)

				// 可以进一步处理获取到的URL
				// ...
			} else {
				log.Println("Failed to parse HTTP layer:", err)
			}
		}
	}
}

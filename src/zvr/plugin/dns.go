package plugin

import (
	"zvr/server"
	"zvr/utils"
	"fmt"
)

const (
	REMOVE_DNS_PATH = "/removedns"
	SET_DNS_PATH = "/setdns"
)

type dnsInfo struct {
	DnsAddress string `json:"dnsAddress"`
	NicMac string `json:"nicMac"`
}

type setDnsCmd struct {
	Dns []dnsInfo `json:"dns"`
}

type removeDnsCmd struct {
	Dns []dnsInfo `json:"dns"`
}


func makeDnsFirewallRuleDescription(nicname string) string {
	return fmt.Sprintf("DNS-for-%s", nicname)
}

func setDnsHandler(ctx *server.CommandContext) interface{} {
	tree := server.NewParserFromShowConfiguration().Tree

	cmd := &setDnsCmd{}
	ctx.GetCommand(cmd)

	dnsByMac := make(map[string][]dnsInfo)
	for _, info := range cmd.Dns {
		dns := dnsByMac[info.NicMac]
		if dns == nil {
			dns = make([]dnsInfo, 0)
		}
		dns = append(dns, info)
		dnsByMac[info.NicMac] = dns
	}

	/* delete previous config */
	tree.Deletef("service dns forwarding")

	/* dns is ordered in management node, should not be changed in vyos */
	for _, info := range cmd.Dns {
		tree.SetfWithoutCheckExisting("service dns forwarding name-server %s", info.DnsAddress)
	}

	for mac, _ := range dnsByMac {
		eth, err := utils.GetNicNameByMac(mac); utils.PanicOnError(err)
		tree.SetfWithoutCheckExisting("service dns forwarding listen-on %s", eth)

		des := makeDnsFirewallRuleDescription(eth)
		if r := tree.FindFirewallRuleByDescription(eth, "local", des); r == nil {
			tree.SetFirewallOnInterface(eth, "local",
				fmt.Sprintf("description %v", des),
				"destination port 53",
				"protocol tcp_udp",
				"action accept",
			)

			tree.AttachFirewallToInterface(eth, "local")
		}
	}

	tree.Apply(false)

	return nil
}

func removeDnsHandler(ctx *server.CommandContext) interface{} {
	tree := server.NewParserFromShowConfiguration().Tree

	cmd := &removeDnsCmd{}
	ctx.GetCommand(cmd)

	for _, info := range cmd.Dns {
		tree.Deletef("service dns forwarding name-server %s", info.DnsAddress)
	}

	tree.Apply(false)
	return nil
}

func DnsEntryPoint() {
	server.RegisterAsyncCommandHandler(SET_DNS_PATH, server.VyosLock(setDnsHandler))
	server.RegisterAsyncCommandHandler(REMOVE_DNS_PATH, server.VyosLock(removeDnsHandler))
}

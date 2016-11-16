package router

//Sanity Check Packet -> minimum length and correct checksum
//decrement TTL and recompute packet checksum (l3 recompute checksum)

//lookup in the longest prefix matching table:
//destination ip address of the packet.

//LONGEST PREFIX MATCHING trivialimplementation

var Router = `
#include <linux/ip.h>
#include <linux/bpf.h>

// #define BPF_TRACE
#undef BPF_TRACE

#define BPF_LOG
// #undef BPF_LOG

#define ROUTING_TABLE_DIM 10
#define ROUTER_PORT_N     10
#define ARP_TABLE_DIM     10

#define IP_TTL_OFFSET  8
#define IP_CSUM_OFFSET 10

#define ETH_DST_OFFSET  0
#define ETH_SRC_OFFSET  6
#define ETH_TYPE_OFFSET 12

/*Routing Table Entry*/
struct rt_entry{
  u32 network;	//network: e.g. 192.168.1.0
  u32 netmask;	//netmask: e.g. 255.255.255.0
  u32 port;			//port of the router
};

/*Router Port*/
struct r_port{
  u32 ip;				//ip addr : e.g. 192.168.1.254
  u32 netmask;	//netmask : e.g. 255.255.255.0
  u64 mac;			//mac addr: e.g. a1:b2:c3:ab:cd:ef
};

/*Arp Table Key*/
struct arp_table_key{
  u32 ip;				//ip addr : e.g. 192.168.1.2
  u32 port;			//port    : e.g. 1
};

/*
  The Routing table is implemented as an array of struct rt_entry (Routing Table Entry)
  the longest prefix matching algorithm (at least a simplified version)
  is implemented performing a bounded loop over the entries of the routing table.
  We assume that the control plane puts entry ordered from the longest netmask
  to the shortest one.
*/
BPF_TABLE("array", u32, struct rt_entry, routing_table, ROUTING_TABLE_DIM);

/*
  Router Port table provides a way to simulate the physical interface of the router
  The ip address is used to answer to the arp request (TO IMPLEMENT)
  The mac address is used as mac_scr for the outcoming packet on that interface,
  and as mac address contained in the arp reply
*/
BPF_TABLE("hash", u32, struct r_port, router_port, ROUTER_PORT_N);

/*
  We shold have an arp table for each port of the router?
  For now we assume to send packet exiting the router interfaces in broadcast
  (mac dst = ff:ff:ff:ff:ff:ff)

  How can we implement multiple arp tables?
  One possible implementation using one single map is the following
  key{ ip + port number } -> value {mac_address}
*/
BPF_TABLE("hash", struct arp_table_key, u64, arp_table, ARP_TABLE_DIM);

static int handle_rx(void *skb, struct metadata *md) {
  u8 *cursor = 0;
  struct ethernet_t *ethernet = cursor_advance(cursor, sizeof(*ethernet));

  #ifdef BPF_TRACE
    bpf_trace_printk("in_ifc:%d\n",md->in_ifc);
    bpf_trace_printk("eth_type:%x mac_scr:%lx mac_dst:%lx\n",ethernet->type,ethernet->src,ethernet->dst);
  #endif

  //TODO
  //sanity check of the packet.
  //if something wrong -> DROP the packet

  struct ip_t *ip = cursor_advance(cursor,sizeof(*ip));

  #ifdef BPF_TRACE
    bpf_trace_printk("ttl:%u ip_scr:%x ip_dst:%x \n",ip->ttl,ip->src,ip->dst);
    // bpf_trace_printk("(before) ttl: %d checksum: %x\n",ip->ttl,ip->hchecksum);
  #endif

  /*
    decrement TTL and recompute packet checksum (l3 recompute checksum).
    if ttl <= 1 DROP the packet.
    eventually send ICMP message for the packet dropped.
    (maybe to avoid for security reasons)
  */

  __u8 old_ttl = ip->ttl;
  __u8 new_ttl;

  if(old_ttl <= 1){
    #ifdef BPF_TRACE
      bpf_trace_printk("packet DROP (ttl <= 1)\n");
    #endif
    return RX_DROP;
  }

  new_ttl = old_ttl - 1;
  bpf_l3_csum_replace(skb, sizeof(*ethernet) + IP_CSUM_OFFSET , old_ttl, new_ttl, sizeof(__u16));
  bpf_skb_store_bytes(skb, sizeof(*ethernet) + IP_TTL_OFFSET , &new_ttl, sizeof(old_ttl), 0);

  #ifdef BPF_TRACE
    // bpf_trace_printk("(after ) ttl: %d checksum: %x\n",ip->ttl,ip->hchecksum);
  #endif

  /*
    ROUTING ALGORITHM (simplified)

    for each item in the routing table (upbounded loop)
    apply the netmask on dst_ip_address
    (possible optimization, not recompute if at next iteration the netmask is the same)
    if masked address == network in the routing table
      1- change src mac to otuput port mac
      2- change dst mac to lookup arp table (or send to fffffffffffff)
      3- forward the packet to dst port
  */

  int i = 0;
  struct rt_entry * rt_entry_p = 0;

  u32 ip_dst_masked = 0;
  u64 new_src_mac = 0;
  u64 new_dst_mac = 0;
  u32 out_port = 0;
  struct r_port * r_port_p = 0;

  //i:0
  rt_entry_p = routing_table.lookup(&i);
  if(rt_entry_p){
    ip_dst_masked = ip->dst & rt_entry_p->netmask;
    if (ip_dst_masked == rt_entry_p->network){
      goto FORWARD;
      }
  }

  //i:1
  i++;
  rt_entry_p = routing_table.lookup(&i);
  if(rt_entry_p){
    ip_dst_masked = ip->dst & rt_entry_p->netmask;
    if (ip_dst_masked == rt_entry_p->network){
      goto FORWARD;
    }
  }

  //i:2
  i++;
  rt_entry_p = routing_table.lookup(&i);
  if(rt_entry_p){
    ip_dst_masked = ip->dst & rt_entry_p->netmask;
    if (ip_dst_masked == rt_entry_p->network){
    goto FORWARD;
    }
  }

  //i:3
  i++;
  rt_entry_p = routing_table.lookup(&i);
  if(rt_entry_p){
    ip_dst_masked = ip->dst & rt_entry_p->netmask;
    if (ip_dst_masked == rt_entry_p->network){
      goto FORWARD;
    }
  }

  //i:4
  i++;
  rt_entry_p = routing_table.lookup(&i);
  if(rt_entry_p){
    ip_dst_masked = ip->dst & rt_entry_p->netmask;
    if (ip_dst_masked == rt_entry_p->network){
      goto FORWARD;
    }
  }

  //i:5
  i++;
  rt_entry_p = routing_table.lookup(&i);
  if(rt_entry_p){
    ip_dst_masked = ip->dst & rt_entry_p->netmask;
    if (ip_dst_masked == rt_entry_p->network){
      goto FORWARD;
    }
  }

  //i:6
  i++;
  rt_entry_p = routing_table.lookup(&i);
  if(rt_entry_p){
    ip_dst_masked = ip->dst & rt_entry_p->netmask;
    if (ip_dst_masked == rt_entry_p->network){
      goto FORWARD;
    }
  }

  //i:7
  i++;
  rt_entry_p = routing_table.lookup(&i);
  if(rt_entry_p){
    ip_dst_masked = ip->dst & rt_entry_p->netmask;
    if (ip_dst_masked == rt_entry_p->network){
      goto FORWARD;
    }
  }

  //i:8
  i++;
  rt_entry_p = routing_table.lookup(&i);
  if(rt_entry_p){
    ip_dst_masked = ip->dst & rt_entry_p->netmask;
    if (ip_dst_masked == rt_entry_p->network){
      goto FORWARD;
    }
  }

  //i:9
  i++;
  rt_entry_p = routing_table.lookup(&i);
  if(rt_entry_p){
    ip_dst_masked = ip->dst & rt_entry_p->netmask;
    if (ip_dst_masked == rt_entry_p->network){
      goto FORWARD;
    }
  }

  DROP:
  #ifdef BPF_LOG
    bpf_trace_printk("in: %d out: -- DROP\n",md->in_ifc);
  #endif
  return RX_DROP;

  FORWARD:
  //Select out interface
  out_port = rt_entry_p->port;
  if (out_port<=0)
    goto DROP;

  #ifdef BPF_LOG
    bpf_trace_printk("ROUTING TABLE MATCH (#%d) network: %x netmask: %x\n", i, ip_dst_masked, rt_entry_p->netmask);
  #endif

  //change src mac
  r_port_p = router_port.lookup(&out_port);
  if(r_port_p){
    new_src_mac = r_port_p->mac;
    bpf_skb_store_bytes(skb,ETH_SRC_OFFSET,&new_src_mac,6,0);
  }

  //change dst mac to ff:ff:ff:ff:ff:ff (TODO arp table)
  new_dst_mac = 0xffffffffffff;
  bpf_skb_store_bytes(skb,ETH_DST_OFFSET,&new_dst_mac,6,0);

  #ifdef BPF_TRACE
    bpf_trace_printk("eth_type:%x mac_scr:%lx mac_dst:%lx\n",ethernet->type,ethernet->src,ethernet->dst);
    bpf_trace_printk("out_ifc: %d\n",out_port);
  #endif

  #ifdef BPF_LOG
    bpf_trace_printk("in: %d out: %d REDIRECT\n",md->in_ifc,out_port);
  #endif

  pkt_redirect(skb,md,out_port);
  return RX_REDIRECT;

}
`
Vim�UnDo� �T�	,�Qw:�	-M0��బ�h
�xn��N5�`                                     Z���    _�                             ����                                                                                                                                                                                                                                                                                                                                                             Z��i     �                   5�_�                           ����                                                                                                                                                                                                                                                                                                                                                             Z��q     �                 import()5�_�                            ����                                                                                                                                                                                                                                                                                                                                                             Z��s     �                5�_�                           ����                                                                                                                                                                                                                                                                                                                                                             Z��u     �               	""5�_�                            ����                                                                                                                                                                                                                                                                                                                                                             Z��v     �                  �               5�_�                           ����                                                                                                                                                                                                                                                                                                                                                             Z��     �                ( //  http://play.golang.org/p/m8TNTtygK05�_�                       
    ����                                                                                                                                                                                                                                                                                                                                                             Z��     �               +func Hosts(cidr string) ([]string, error) {5�_�      	                     ����                                                                                                                                                                                                                                                                                                                                                             Z��     �               	 �             5�_�      
           	      	    ����                                                                                                                                                                                                                                                                                                                                                             Z��     �               
	 if len()5�_�   	              
          ����                                                                                                                                                                                                                                                                                                                                                             Z��     �               	 if len(ips)5�_�   
                        ����                                                                                                                                                                                                                                                                                                                                                             Z���     �               	 if len(ips) <3{}5�_�                           ����                                                                                                                                                                                                                                                                                                                                                             Z���     �               	 5�_�                           ����                                                                                                                                                                                                                                                                                                                                                             Z���     �             �             5�_�                           ����                                                                                                                                                                                                                                                                                                                                                             Z���     �                	 	5�_�                           ����                                                                                                                                                                                                                                                                                                                                                             Z���     �               "     return ips[0 : len(ips)], nil5�_�                           ����                                                                                                                                                                                                                                                                                                                                                             Z���     �               #     	return ips[0 : len(ips)], nil5�_�                           ����                                                                                                                                                                                                                                                                                                                                                             Z���     �               #     	return ips[1 : len(ips)], nil5�_�                           ����                                                                                                                                                                                                                                                                                                                                                             Z���     �               "     return ips[0 : len(ips)], nil5�_�                           ����                                                                                                                                                                                                                                                                                                                                                             Z���     �               "     return ips[1 : len(ips)], nil5�_�                            ����                                                                                                                                                                                                                                                                                                                                                             Z���    �                  package znet       import(   	"net"   )       -func Iprange(cidr string) ([]string, error) {   *     ip, ipnet, err := net.ParseCIDR(cidr)        if err != nil {            return nil, err        }            var ips []string   A     for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {   '         ips = append(ips, ip.String())        }         // remove broadcast address   	 if len(ips) <3{   #     	return ips[0 : len(ips)], nil   	 }   $     return ips[1 : len(ips)-1], nil    }        func inc(ip net.IP) {   (     for j := len(ip) - 1; j >= 0; j-- {            ip[j]++            if ip[j] > 0 {                break   
         }        }    }5�_�                           ����                                                                                                                                                                                                                                                                                                                                                             Z���     �                	// remove broadcast address5�_�                           ����                                                                                                                                                                                                                                                                                                                                                             Z���     �             �             5�_�                            ����                                                                                                                                                                                                                                                                                                                                                             Z���    �                  package znet       import (   	"net"   )       -func Iprange(cidr string) ([]string, error) {   &	ip, ipnet, err := net.ParseCIDR(cidr)   	if err != nil {   		return nil, err   	}       	var ips []string   =	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {    		ips = append(ips, ip.String())   	}   	if len(ips) < 3 {   		return ips[0:len(ips)], nil   	}   	// remove broadcast address    	return ips[1 : len(ips)-1], nil   }       func inc(ip net.IP) {   $	for j := len(ip) - 1; j >= 0; j-- {   			ip[j]++   		if ip[j] > 0 {   			break   		}   	}   }5��
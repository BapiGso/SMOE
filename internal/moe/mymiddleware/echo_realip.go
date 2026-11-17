package mymiddleware

import (
	"fmt"
	"net"

	"github.com/labstack/echo/v5"
)

// IPExtractor 决定 c.RealIP() 的来源，评论、点赞限流和浏览量去重都按它区分访客。
// 没有可信代理时只认连接的对端地址，请求头一概不信；
// 有的话从 X-Forwarded-For 末尾往前跳过可信代理，取第一个不可信的地址，伪造的头部前缀不起作用。
// proxies 每项是 IP 或 CIDR。
func IPExtractor(proxies []string) (echo.IPExtractor, error) {
	if len(proxies) == 0 {
		return echo.ExtractIPDirect(), nil
	}
	// 只信配置里写的，不沿用 echo 默认信任的回环和内网地址
	opts := []echo.TrustOption{echo.TrustLoopback(false), echo.TrustLinkLocal(false), echo.TrustPrivateNet(false)}
	for _, p := range proxies {
		_, ipNet, err := net.ParseCIDR(p)
		if err != nil {
			ip := net.ParseIP(p)
			if ip == nil {
				return nil, fmt.Errorf("%q 不是 IP 或 CIDR", p)
			}
			if ip4 := ip.To4(); ip4 != nil {
				ip = ip4
			}
			ipNet = &net.IPNet{IP: ip, Mask: net.CIDRMask(len(ip)*8, len(ip)*8)}
		}
		opts = append(opts, echo.TrustIPRange(ipNet))
	}
	return echo.ExtractIPFromXFFHeader(opts...), nil
}

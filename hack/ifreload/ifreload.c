// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

#include <netlink/route/qdisc.h>
#include <net/if.h>
#include <sys/ioctl.h>
#include <errno.h>
#include <stdarg.h>
#include <stdbool.h>
#include <time.h>
#include <unistd.h>


static const char *LINK_NAME = "eth0";

static const int BOUNCE_INTERVAL = 600;
static const int HWM = 10240 * 90 / 100;
static const int HWM_COUNT = 10;
static const int INITIAL_SLEEP = 3 * 3600;
static const int INTERVAL = 60;

static int sock;


static void __logger(const char *format, va_list args) {
	time_t now = time(NULL);

	char buf[64];
	strftime(buf, sizeof(buf), "%Y/%m/%d %H:%M:%S %Z", localtime(&now));
	fprintf(stderr, "%s: ", buf);

	vfprintf(stderr, format, args);
}


static void logger(const char *format, ...) {
	va_list args;

	va_start(args, format);
	__logger(format, args);
	va_end(args);

	fputc('\n', stderr);
}


static void logger_error(const char *format, ...) {
	va_list args;

	va_start(args, format);
	__logger(format, args);
	va_end(args);

	fprintf(stderr, ": %s\n", strerror(errno));
}


static int ifupdown(bool up) {
	struct ifreq ifreq;
	memset(&ifreq, 0, sizeof(ifreq));

	strncpy(ifreq.ifr_name, LINK_NAME, IFNAMSIZ);

	if (ioctl(sock, SIOCGIFFLAGS, &ifreq)) {
		logger_error("ioctl SIOCGIFFLAGS");
		return -1;
	}

	if (up) {
		logger("bringing up interface %s", LINK_NAME);
		ifreq.ifr_flags |= IFF_UP;

	} else {
		logger("bringing down interface %s", LINK_NAME);
		ifreq.ifr_flags &= ~IFF_UP;
	}

	if (ioctl(sock, SIOCSIFFLAGS, &ifreq)) {
		logger_error("ioctl SIOCSIFFLAGS");
		return -1;
	}

	sleep(5);

	logger("done");

	return 0;
}


static void cb(struct nl_object *obj, void *arg) {
	struct rtnl_qdisc *qdisc = nl_object_priv(obj);
	struct rtnl_tc *tc = TC_CAST(qdisc);

	*(uint64_t *)arg = rtnl_tc_get_stat(tc, RTNL_TC_QLEN);
}


int main(int argc, char **argv) {
	if (argc > 1) {
		logger("dry run");
	} else if (getuid()) {
		logger("%s: must run as root", argv[0]);
		return 1;
	}

	if ((sock = socket(AF_INET, SOCK_DGRAM, 0)) < 0) {
		perror("socket");
		return 1;
	}

	struct nl_sock *nl_sock = nl_socket_alloc();
	if (!nl_sock) {
		logger("nl_socket_alloc failed");
		return 1;
	}

	int err;

	if ((err = nl_connect(nl_sock, NETLINK_ROUTE))) {
		logger("nl_socket_alloc failed: %d", err);
		return 1;
	}

	struct nl_cache *link_cache, *qdisc_cache;

	if ((err = rtnl_qdisc_alloc_cache(nl_sock, &qdisc_cache))) {
		logger("rtnl_qdisc_alloc_cache failed: %d", err);
		return 1;
	}

	if ((err = rtnl_link_alloc_cache(nl_sock, AF_UNSPEC, &link_cache))) {
		logger("rtnl_link_alloc_cache failed: %d", err);
		return 1;
	}

	struct rtnl_link *link = rtnl_link_get_by_name(link_cache, LINK_NAME);
	if (!link) {
		logger("rtnl_link_get_by_name failed: link %s not found", LINK_NAME);
		return 1;
	}

	struct rtnl_qdisc *qdisc = rtnl_qdisc_alloc();
	if (!qdisc) {
		logger("rtnl_qdisc_alloc failed");
		return 1;
	}

	struct rtnl_tc *tc = TC_CAST(qdisc);

	rtnl_tc_set_link(tc, link);
	rtnl_tc_set_parent(tc, TC_H_ROOT);

	time_t last_bounce = 0;
	int count = 0;

	uint64_t qlen = 0;

	nl_cache_foreach_filter(qdisc_cache, OBJ_CAST(qdisc), &cb, &qlen);

	logger("queue length: %ld", qlen);

	if (argc > 1) {
		return 0;
	}

	logger("sleeping");

	sleep(INITIAL_SLEEP);

	logger("running");

	while (1) {
		sleep(INTERVAL);

		if ((err = nl_cache_refill(nl_sock, qdisc_cache))) {
			logger("nl_cache_refill failed: %d", err);
			continue;
		}

		qlen = 0;

		nl_cache_foreach_filter(qdisc_cache, OBJ_CAST(qdisc), &cb, &qlen);

		if (qlen) {
			logger("queue length: %ld", qlen);
		}

		if (qlen > HWM) {
			count++;
		} else {
			count = 0;
		}

		if (count < HWM_COUNT) {
			continue;
		}

		logger("detected %d consecutive queue full events", count);

		time_t now = time(NULL);
		if (now - last_bounce < BOUNCE_INTERVAL) {
			continue;
		}

		ifupdown(false);

		while (ifupdown(true));

		last_bounce = now;
		count = 0;
	}

	return 0;
}

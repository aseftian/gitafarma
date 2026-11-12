self.addEventListener("push", (event) => {
  if (!event.data) return;

  let payload = {
    title: "Update Harga Obat",
    body: "Ada pembaruan harga obat di Apotek Gita Farma.",
    url: "/",
  };

  try {
    const parsed = event.data.json();
    payload = { ...payload, ...parsed };
  } catch {
    payload.body = event.data.text() || payload.body;
  }

  event.waitUntil(
    self.registration.showNotification(payload.title, {
      body: payload.body,
      icon: "/icons/icon-192.svg",
      badge: "/icons/icon-192.svg",
      data: { url: payload.url || "/" },
    }),
  );
});

function resolveSafeSameOriginUrl(rawUrl) {
  try {
    const resolved = new URL(rawUrl || "/", self.location.origin);
    if (resolved.origin !== self.location.origin) {
      return "/";
    }
    return resolved.pathname + resolved.search + resolved.hash;
  } catch {
    return "/";
  }
}

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const targetUrl = resolveSafeSameOriginUrl(
    event.notification.data && event.notification.data.url,
  );

  event.waitUntil(
    self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((clients) => {
      for (const client of clients) {
        if ("focus" in client) {
          client.navigate(targetUrl);
          return client.focus();
        }
      }
      if (self.clients.openWindow) {
        return self.clients.openWindow(targetUrl);
      }
      return undefined;
    }),
  );
});

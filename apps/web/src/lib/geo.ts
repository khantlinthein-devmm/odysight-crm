/** A phone GPS fix sent with cleaner check-in/out. */
export interface GeoFix {
  latitude: number;
  longitude: number;
  /** Error radius in metres, as reported by the device. */
  accuracy: number;
}

/**
 * Current position, or null when the browser has no geolocation, the user
 * denies it, or no fix arrives in time. Never throws.
 */
export function getPosition(timeoutMs = 12000): Promise<GeoFix | null> {
  if (typeof navigator === "undefined" || !navigator.geolocation) return Promise.resolve(null);
  return new Promise((resolve) => {
    navigator.geolocation.getCurrentPosition(
      (pos) =>
        resolve({
          latitude: pos.coords.latitude,
          longitude: pos.coords.longitude,
          accuracy: pos.coords.accuracy,
        }),
      () => resolve(null),
      // maximumAge 0: after walking to the site, a retry must not reuse the
      // earlier (far-away) fix.
      { enableHighAccuracy: true, timeout: timeoutMs, maximumAge: 0 },
    );
  });
}

/**
 * Reads "lat,lng" from free text or a Google Maps link (…/@13.74,100.53,17z,
 * …?q=13.74,100.53, …!3d13.74!4d100.53). Returns null when none is found.
 */
export function parseCoordinates(text: string): { latitude: number; longitude: number } | null {
  const s = text.trim();
  const patterns = [
    /!3d(-?\d+(?:\.\d+)?)!4d(-?\d+(?:\.\d+)?)/,
    /@(-?\d+(?:\.\d+)?),\s*(-?\d+(?:\.\d+)?)/,
    /[?&](?:q|query|ll)=(-?\d+(?:\.\d+)?),\s*(-?\d+(?:\.\d+)?)/,
    /^(-?\d+(?:\.\d+)?)\s*,\s*(-?\d+(?:\.\d+)?)$/,
  ];
  for (const re of patterns) {
    const m = s.match(re);
    if (m) {
      const latitude = Number(m[1]);
      const longitude = Number(m[2]);
      if (Math.abs(latitude) <= 90 && Math.abs(longitude) <= 180) return { latitude, longitude };
    }
  }
  return null;
}

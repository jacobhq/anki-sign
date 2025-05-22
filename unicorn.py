import requests
from datetime import datetime, timedelta
import unicornhatmini
import time
import sys

ANKI_HOST = "DESKTOP-028KM9I.local"
DAYS = 119
BRIGHTNESS = 1

COLORS = [
    (0, 0, 0),
    (144, 238, 144),
    (0, 200, 0),
    (0, 128, 0),
    (0, 100, 0),
    (0, 64, 0),
]

def get_daily_review_counts(days=DAYS):
    try:
        payload = {
            "action": "getNumCardsReviewedByDay",
            "version": 6
        }
        response = requests.post(f"http://{ANKI_HOST}:8765", json=payload, timeout=5)
        response.raise_for_status()
        raw_data = response.json().get('result', [])
        print(raw_data)

        review_map = {
            datetime.strptime(day, "%Y-%m-%d").date(): int(count)
            for day, count in raw_data
        }

        today = datetime.now().date()
        history = []
        for i in range(days - 1, -1, -1):
            date = today - timedelta(days=i)
            history.append(review_map.get(date, 0))
        return history
    except Exception as e:
        print("Failed to fetch review data:", e)
        return [0] * days

def map_to_color_buckets(counts):
    max_count = max(counts) or 1
    buckets = []
    for count in counts:
        if count == 0:
            buckets.append(0)
        else:
            level = int(5 * count / max_count)
            buckets.append(min(level, 5))
    return buckets

def draw_heatmap(buckets):
    uhm = unicornhatmini.UnicornHATMini()
    uhm.set_brightness(BRIGHTNESS)
    uhm.clear()

    for i, level in enumerate(buckets):
        x = i // 7
        y = i % 7
        if x < 17:
            r, g, b = COLORS[level]
            uhm.set_pixel(x, y, r, g, b)
    uhm.show()

if __name__ == "__main__":
    counts = get_daily_review_counts()
    buckets = map_to_color_buckets(counts)
    draw_heatmap(buckets)

    try:
        print("Heatmap displayed. Press Ctrl+C to exit.")
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("Exiting and clearing display.")
        uhm = unicornhatmini.UnicornHATMini()
        uhm.clear()
        uhm.show()
        sys.exit(0)

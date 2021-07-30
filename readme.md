# Gemini_Exchange_Exporter
This is a prometheus exporter for the gemini exchange using their public APIs. 
This goes through and grabs all the symbols on their exchange and then exposes the following:

opening_price = "Open price from 24 hours ago (per Currency)",
high_price = "High price from 24 hours ago (per Currency).",
low_price = "Low price from 24 hours ago (per Currency).",
close_price = "Close price (most recent trade)(per Currency).",
bid_price = "Current best bid (per Currency).",
ask_price = "Current best offer (per Currency)."
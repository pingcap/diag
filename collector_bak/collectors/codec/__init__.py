#coding: utf8
from raw import RawEncoder, RawDecoder
from json_codec import JSONEncoder, JSONDecoder

# coding: utf8
class Codec:
    def __init__(self, encoder, decoder):
        self.encoder = encoder
        self.decoder = decoder

    def encode(self, obj):
        return self.encoder.encode(obj)

    def decode(self, data):
        return self.decoder.decode(data)

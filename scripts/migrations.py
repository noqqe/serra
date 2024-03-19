from pymongo import MongoClient
import os
import pymongo

CONNECTION_STRING = os.getenv("MONGODB_URI")
client = MongoClient(CONNECTION_STRING+'/admin')


# Create a new collection
collection = client["serra"]["cards"]

cards=collection.find()

for c in cards:
    print(c["_id"])
    f = { '_id': c["_id"] }
    u = { "$set": { 'collectornumber': str(c["collectornumber"])} }
    print("%s %s %s" % (f, u, c["collectornumber"]))
    collection.update_one(f, u)

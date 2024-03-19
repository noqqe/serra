from pymongo import MongoClient
import os
import pymongo

CONNECTION_STRING = os.getenv("MONGODB_URI")
client = MongoClient(CONNECTION_STRING+'/admin')


# Create a new collection
collection = client["serra"]["cards"]

cards=collection.find()

for c in cards:
    print("./serra add %s/%s -c %s" % (c["set"], c["collectornumber"], c["serra_count"]))

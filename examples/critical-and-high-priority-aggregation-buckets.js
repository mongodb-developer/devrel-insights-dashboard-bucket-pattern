db.alerts.dashboard.deleteOne({_id: "priority"});

db.alerts.aggregate([
    {
        "$match": {
            "cleared": false,
            "priority": {"$in": ["Critical", "High"]}
        }
    },
    {
        "$setWindowFields": {
            "sortBy": {"createdAt": 1},
            "output": {
                "index": {
                    "$documentNumber": {}
                }
            }
        }
    },
    {
        "$addFields": {
            "bucket": {"$floor": {"$divide": [{"$subtract": ["$index", 1]}, 5000]}}
        }
    },
    {
        "$group": {
            "_id": "$bucket",
            "values": {
                "$push": {
                    "name": "$name",
                    "priority": "$priority",
                    "createdAt": "$createdAt",
                    "cleared": "$cleared"
                }
            },
            "count": {"$sum": 1}
        }
    },
    {
        "$project": {
            "_id": {"$concat": ["priority_bucket_", {"$toString": "$_id"}]},
            "values": 1,
            "count": 1
        }
    },
    {
        "$merge": {
            "into": "dashboard",
            "whenMatched": "replace",
            "whenNotMatched": "insert"
        }
    }
]);
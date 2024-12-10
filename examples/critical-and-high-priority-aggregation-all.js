db.alerts.aggregate([
    {
        $match: {
            cleared: false,
            priority: {
                $in: ["Critical", "High"]
            }
        }
    },
    {
        $sort: {
            createdAt: -1
        }
    },
    {
        $group: {
            _id: "priority",
            count: {
                $sum: 1
            },
            values: {
                $push: "$$ROOT"
            }
        }
    },
    {
        $merge:
            {
                into: "dashboard"
            }
    }
])
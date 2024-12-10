db.alerts.aggregate([
    {
        $match: {
            cleared: false
        }
    },
    {
        $sort: {
            createdAt: -1
        }
    },
    {
        $limit: 25
    },
    {
        $group: {
            _id: "top25",
            values: {
                $push: "$$ROOT"
            }
        }
    },
    {
        $out: "dashboard"
    }
])

// Connect to the desired database
const alertdb = db.getSiblingDB("alertdb");

// Define the collection
const collection = alertdb.alerts;

// Clear the collection if it already exists
collection.drop();

// Helper function to generate a random date within the last 365 days
function getRandomDate() {
    const now = new Date();
    const past = new Date(now);

    past.setDate(now.getDate() - 365);
    return new Date(past.getTime() + Math.random() * (now.getTime() - past.getTime()));
}
// Helper function to get a random priority with specified probabilities
function getRandomPriority() {
    const rand = Math.random();

    if (rand < 0.70) {
        return "Low";
    } else if (rand < 0.90) {
        return "Medium";
    } else if (rand < 0.999) {
        return "High";
    } else {
        return "Critical";
    }
}

// Helper function to get a random cleared status
function getRandomCleared() {
    return Math.random() < 0.8;
}

// Insert 10,000,000 sample documents
let bulk = collection.initializeUnorderedBulkOp();
for (let i = 0; i < 5000000; i++) {
    bulk.insert({
        name: `Alert ${i + 1}`,
        priority: getRandomPriority(),
        createdAt: getRandomDate(),
        cleared: getRandomCleared()
    });

    if(i % 50000) {
        bulk.execute();
        bulk = collection.initializeUnorderedBulkOp();
    }
}

bulk.execute();
print("Inserted 5,000,000 sample documents into the 'alerts' collection.");
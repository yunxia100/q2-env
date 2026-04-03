// MongoDB init
db = db.getSiblingDB('q2_db');

var collections = [
    'user',
    'custservice',
    'proxy',
    'proxy_extractor',
    'device',
    'device_label',
    'device_batch',
    'device_friend',
    'device_material',
    'task',
    'task_greet_rule',
    'task_greet_word',
    'task_remark_rule',
    'task_usedb',
    'task_srcdb',
    'task_materialdb',
    'task_realinfodb',
    'group_task',
    'group_material',
    'risk_management',
    'webdevice'
];

collections.forEach(function(name) {
    if (!db.getCollectionNames().includes(name)) {
        db.createCollection(name);
        print('Created: ' + name);
    }
});

db.user.createIndex({ "username": 1 }, { unique: true, sparse: true });
db.device.createIndex({ "uin": 1 });
db.device.createIndex({ "user_id": 1 });
db.device.createIndex({ "batch_id": 1 });
db.device_label.createIndex({ "user_id": 1 });
db.device_batch.createIndex({ "user_id": 1 });
db.task.createIndex({ "user_id": 1 });
db.task.createIndex({ "mode": 1 });
db.group_task.createIndex({ "user_id": 1 });
db.group_material.createIndex({ "user_id": 1 });
db.risk_management.createIndex({ "user_id": 1 });
db.proxy.createIndex({ "region": 1 });
db.webdevice.createIndex({ "uin": 1 });

print('=== init complete ===');

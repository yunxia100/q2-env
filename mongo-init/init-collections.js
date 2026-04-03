// ============================================================
// MongoDB 初始化脚本
// 自动创建 ymlink_q2 数据库及所有集合和索引
// ============================================================

db = db.getSiblingDB('ymlink_q2');

// 创建所有集合
var collections = [
    'user',
    'custservice',
    'proxy',
    'proxy_extractor',
    'robot',
    'robot_label',
    'robot_batch',
    'robot_friend',
    'robot_material',
    'task',
    'task_greet_rule',
    'task_greet_word',
    'task_qzone_remark_rule',
    'task_usedb',
    'task_qzonedb',
    'task_materialdb',
    'task_realinfodb',
    'group_task',
    'group_material',
    'risk_management',
    'webrobot'
];

collections.forEach(function(name) {
    if (!db.getCollectionNames().includes(name)) {
        db.createCollection(name);
        print('Created collection: ' + name);
    }
});

// 建议索引
db.user.createIndex({ "username": 1 }, { unique: true, sparse: true });
db.robot.createIndex({ "uin": 1 });
db.robot.createIndex({ "user_id": 1 });
db.robot.createIndex({ "batch_id": 1 });
db.robot_label.createIndex({ "user_id": 1 });
db.robot_batch.createIndex({ "user_id": 1 });
db.task.createIndex({ "user_id": 1 });
db.task.createIndex({ "mode": 1 });
db.group_task.createIndex({ "user_id": 1 });
db.group_material.createIndex({ "user_id": 1 });
db.risk_management.createIndex({ "user_id": 1 });
db.proxy.createIndex({ "region": 1 });
db.webrobot.createIndex({ "uin": 1 });

print('=== MongoDB initialization complete ===');

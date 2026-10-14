\connect scheduler_db

-- scheduler-service's snapshot of dealership data (migrations/00002), until
-- dealership-service exists to publish it. Ids stand in for
-- dealership_db's dealership / service_bay / service ids.

INSERT INTO hub (id, name, address, open_hour, close_hour, timezone) VALUES
  (1, 'Service Bay District 1', '12 Nguyen Hue, District 1, Ho Chi Minh City', '08:00', '18:00', 'Asia/Ho_Chi_Minh'),
  (2, 'Service Bay Thu Duc',    '88 Vo Van Ngan, Thu Duc, Ho Chi Minh City',   '07:30', '17:30', 'Asia/Ho_Chi_Minh'),
  (3, 'Service Bay Cau Giay',   '25 Xuan Thuy, Cau Giay, Hanoi',               '08:00', '17:00', 'Asia/Ho_Chi_Minh');

INSERT INTO hub_bay (id, hub_id, code, status) VALUES
  (101, 1, 'BAY-01', 'AVAILABLE'),
  (102, 1, 'BAY-02', 'AVAILABLE'),
  (103, 1, 'BAY-03', 'AVAILABLE'),
  (201, 2, 'BAY-01', 'AVAILABLE'),
  (202, 2, 'BAY-02', 'AVAILABLE'),
  (203, 2, 'BAY-03', 'MAINTENANCE'),
  (301, 3, 'BAY-01', 'AVAILABLE'),
  (302, 3, 'BAY-02', 'AVAILABLE');

INSERT INTO catalog_service (id, name, service_type, fee, duration_minutes) VALUES
  (1, 'Oil and filter change',   'Maintenance', 45.00,  30),
  (2, 'Tire rotation',           'Maintenance', 25.00,  30),
  (3, 'Brake inspection',        'Inspection',  60.50,  60),
  (4, 'Battery test',            'Inspection',  20.00,  30),
  (5, 'Air conditioning service', 'Repair',     75.00,  60),
  (6, 'Full periodic service',   'Maintenance', 180.00, 120),
  (7, 'Warranty inspection',     'Warranty',    0.00,   45);

-- 000036: default checklist templates.
-- Production started with no templates, so "New from template" had nothing
-- to copy. Seed one bilingual template per built-in service type plus a
-- general fallback. A service type that already has a template is skipped,
-- so this never duplicates templates an admin created.

DO $$
DECLARE
    tpl RECORD;
    tid BIGINT;
    i INT;
BEGIN
    FOR tpl IN
        SELECT * FROM (VALUES
            ('', 'General cleaning / ทำความสะอาดทั่วไป', ARRAY[
                'กวาดและถูพื้น / Floors swept and mopped',
                'เช็ดฝุ่นเฟอร์นิเจอร์และชั้นวาง / Furniture and shelves dusted',
                'ทำความสะอาดห้องน้ำ / Bathroom cleaned',
                'เช็ดกระจกและกระจกเงา / Glass and mirrors wiped',
                'ทิ้งขยะ / Rubbish removed',
                'ตรวจความเรียบร้อยกับลูกค้า / Walk-through with customer']),
            ('house_cleaning', 'House cleaning / ทำความสะอาดบ้าน', ARRAY[
                'กวาดและถูพื้นทุกห้อง / All floors swept and mopped',
                'เช็ดฝุ่นเฟอร์นิเจอร์ ชั้นวาง ขอบหน้าต่าง / Furniture, shelves, window sills dusted',
                'ห้องครัว: เคาน์เตอร์ อ่างล้างจาน เตา / Kitchen: counters, sink, stove',
                'ห้องน้ำ: ชักโครก อ่างล้างหน้า ฝักบัว พื้น / Bathroom: toilet, basin, shower, floor',
                'ห้องนอน: จัดเตียง ดูดฝุ่น / Bedrooms: beds made, vacuumed',
                'เช็ดกระจกและกระจกเงา / Glass and mirrors wiped',
                'เช็ดสวิตช์และมือจับประตู / Switches and door handles wiped',
                'ทิ้งขยะและเปลี่ยนถุงขยะ / Rubbish removed, bins relined',
                'ตรวจความเรียบร้อยกับลูกค้า / Walk-through with customer']),
            ('condo_cleaning', 'Condo cleaning / ทำความสะอาดคอนโด', ARRAY[
                'กวาดและถูพื้น / Floors swept and mopped',
                'เช็ดฝุ่นเฟอร์นิเจอร์และชั้นวาง / Furniture and shelves dusted',
                'ห้องครัว: เคาน์เตอร์ อ่างล้างจาน / Kitchen: counters and sink',
                'ห้องน้ำ: ชักโครก อ่างล้างหน้า ฝักบัว / Bathroom: toilet, basin, shower',
                'ระเบียง: กวาดและเช็ดราว / Balcony swept, railing wiped',
                'เช็ดกระจกและประตูระเบียง / Glass and balcony door wiped',
                'ทิ้งขยะ / Rubbish removed',
                'ตรวจความเรียบร้อยกับลูกค้า / Walk-through with customer']),
            ('deep_cleaning', 'Deep cleaning / ทำความสะอาดใหญ่', ARRAY[
                'ขัดพื้นและยาแนว / Floors and grout scrubbed',
                'เช็ดผนัง ขอบบัว ประตู / Walls, skirting boards, doors wiped',
                'ห้องครัว: ขจัดคราบมัน ตู้ครัว เครื่องดูดควัน / Kitchen: degreased, cabinets, hood',
                'ห้องน้ำ: ขัดคราบหินปูนและเชื้อรา / Bathroom: limescale and mould removed',
                'ทำความสะอาดใต้และหลังเฟอร์นิเจอร์ / Under and behind furniture',
                'เช็ดโคมไฟ พัดลม ช่องแอร์ / Lights, fans, air vents wiped',
                'ล้างกระจกและรางหน้าต่าง / Windows and tracks washed',
                'ถ่ายรูปก่อน/หลัง / Before and after photos taken',
                'ตรวจความเรียบร้อยกับลูกค้า / Walk-through with customer']),
            ('move_in_out', 'Move in / out / ทำความสะอาดย้ายเข้า-ออก', ARRAY[
                'ตู้และลิ้นชักทั้งหมด ด้านในและนอก / All cabinets and drawers inside and out',
                'ห้องครัวและเครื่องใช้ไฟฟ้า (ตู้เย็น เตาอบ) / Kitchen and appliances (fridge, oven)',
                'ห้องน้ำขัดทั้งหมด / Bathrooms fully scrubbed',
                'ผนัง สวิตช์ ปลั๊ก / Walls, switches, sockets',
                'หน้าต่าง รางหน้าต่าง มุ้งลวด / Windows, tracks, screens',
                'พื้นขัดและถู / Floors scrubbed and mopped',
                'ระเบียงและทางเข้า / Balcony and entrance',
                'ถ่ายรูปทุกห้อง / Photos of every room',
                'ตรวจความเรียบร้อยกับลูกค้า / Walk-through with customer']),
            ('after_renovation', 'After renovation / ทำความสะอาดหลังรีโนเวท', ARRAY[
                'เก็บเศษวัสดุก่อสร้าง / Construction debris removed',
                'ดูดฝุ่นปูนทุกพื้นผิว / Fine dust vacuumed from all surfaces',
                'ขจัดคราบสี ปูน กาว / Paint, cement, glue stains removed',
                'ล้างกระจกและกรอบหน้าต่าง / Glass and window frames washed',
                'ทำความสะอาดตู้ ชั้น ลิ้นชัก / Cabinets, shelves, drawers cleaned',
                'ห้องน้ำและสุขภัณฑ์ / Bathrooms and fixtures',
                'ขัดและถูพื้น 2 รอบ / Floors scrubbed and mopped twice',
                'ถ่ายรูปก่อน/หลัง / Before and after photos taken',
                'ตรวจความเรียบร้อยกับลูกค้า / Walk-through with customer']),
            ('office_cleaning', 'Office cleaning / ทำความสะอาดสำนักงาน', ARRAY[
                'โต๊ะทำงานและเก้าอี้ / Desks and chairs wiped',
                'ห้องประชุม / Meeting rooms',
                'แพนทรี: อ่างล้างจาน ไมโครเวฟ ตู้เย็น / Pantry: sink, microwave, fridge',
                'ห้องน้ำและเติมสบู่/กระดาษ / Toilets cleaned, soap and paper refilled',
                'พื้น: ดูดฝุ่นพรมและถูพื้น / Floors: carpets vacuumed, hard floors mopped',
                'กระจกทางเข้าและผนังกระจก / Entrance and glass partitions',
                'ทิ้งขยะและขยะรีไซเคิล / General and recycling waste emptied',
                'ตรวจความเรียบร้อยกับผู้ดูแล / Walk-through with site contact']),
            ('junk_removal', 'Junk removal / ขนย้ายขยะ', ARRAY[
                'ยืนยันรายการที่ต้องขนกับลูกค้า / Items confirmed with customer',
                'ถ่ายรูปก่อนขน / Photo before removal',
                'ขนย้ายของทั้งหมด / All items removed',
                'กวาดพื้นที่หลังขน / Area swept after removal',
                'ถ่ายรูปหลังขน / Photo after removal']),
            ('aircon_service', 'Aircon service / ล้างแอร์', ARRAY[
                'ตรวจสภาพและทดสอบก่อนล้าง / Unit checked and tested before',
                'ล้างแผ่นกรองฝุ่น / Filters washed',
                'ล้างคอยล์เย็นและถาดน้ำทิ้ง / Evaporator coil and drain tray cleaned',
                'ล้างคอยล์ร้อน (คอมเพรสเซอร์) / Condenser (outdoor unit) cleaned',
                'ตรวจท่อน้ำทิ้งไม่อุดตัน / Drain pipe clear',
                'ทดสอบความเย็นหลังล้าง / Cooling tested after',
                'เก็บกวาดพื้นที่ทำงาน / Work area cleaned up'])
        ) AS t(service_type, name, items)
    LOOP
        IF NOT EXISTS (SELECT 1 FROM checklist_templates WHERE service_type = tpl.service_type) THEN
            INSERT INTO checklist_templates (name, service_type) VALUES (tpl.name, tpl.service_type)
            RETURNING id INTO tid;
            FOR i IN 1 .. array_length(tpl.items, 1) LOOP
                INSERT INTO checklist_template_items (template_id, label, sort_order)
                VALUES (tid, tpl.items[i], i - 1);
            END LOOP;
        END IF;
    END LOOP;
END $$;

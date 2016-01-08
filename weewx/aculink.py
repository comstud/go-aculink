import contextlib
import datetime
import MySQLdb
import MySQLdb.cursors
import syslog
import time

import weewx
import weewx.drivers

def loader(config_dict, engine):
    aculink_conf = config_dict.get('AcuLink', {})
    user = aculink_conf.get('db_username', 'aculink')
    passwd = aculink_conf.get('db_password', 'aculink')
    db = aculink_conf.get('db_name', 'aculink')
#    host = aculink_conf.get('db_host', None)

    db = MySQLdb.connect(
        user=user,
        passwd=passwd,
        db=db,
        cursorclass=MySQLdb.cursors.DictCursor
    )

    return Aculink(db=db, config=aculink_conf)

def log(msg):
    syslog.syslog(syslog.LOG_INFO, msg)

class Aculink(weewx.drivers.AbstractDevice):
    def __init__(self, db, config):
        self._db = db
        self._archive_interval = 60
        self._epoch_start = datetime.datetime.utcfromtimestamp(0)
        self._last_row = None
        self._last_packet = None
        self._setup_from_config(config)

    def _setup_from_config(self, config):
        self._hardware_name = config.get('hardware_name', 'Acurite 5N1')
        self.out_sensor = config.get('out_sensor', None)
        self.in_sensor = config.get('in_sensor', None)
        self.extra_sensors = {}
        i = 1
        while True:
            extra = config.get('extra_sensor' + str(i), None)
            if extra is None:
                break
            self.extra_sensors[extra] = str(i)
            i += 1

    def _dt_to_epoch(self, dt):
        return (dt - self._epoch_start).total_seconds()

    @contextlib.contextmanager
    def _db_cursor(self):
        try:
            yield self._db.cursor()
            self._db.commit()
        except Exception as err:
            self._db.rollback()
            raise err

    def _select_from_db(self, where, *args, **kwargs):
        qry = 'SELECT * from data ' + where
        if len(args):
            query_arg = args
        elif len(kwargs):
            query_arg = kwargs
        else:
            query_arg = ()
#        log("Running query: " + qry + " " + repr(query_arg))
        with self._db_cursor() as c:
            c.execute(qry, query_arg)
            return c.fetchall()

    def _get_rows_after_dt(self, dt):
        return self._select_from_db(
            'WHERE timestamp > %s ORDER BY timestamp ASC, id ASC',
            dt
        )

    def _get_rows_after_row(self, row):
        return self._select_from_db(
            'WHERE id > %(id)s AND `timestamp` >= %(timestamp)s ' +
            'ORDER BY TIMESTAMP ASC, ID ASC',
            **row
        )

    def _update_packet_for_bridge(self, row, packet):
        if row['pressure_pa'] is not None:
            packet['barometer'] = float(row['pressure_pa']) / 100.0

    def _update_packet_for_extra(self, row, packet):
        extra_num = self.extra_sensors[row['sensor']]
        if row['temperature_c'] is not None:
            packet['extraTemp' + extra_num] = float(row['temperature_c'])
        if row['humidity'] is not None:
            packet['extraHumid' + extra_num] = float(row['humidity'])

    def _update_packet_for_indoor(self, row, packet):
        if row['temperature_c'] is not None:
            packet['inTemp'] = float(row['temperature_c'])
        if row['humidity'] is not None:
            packet['inHumidity'] = float(row['humidity'])

    def _update_packet_for_outdoor(self, row, packet):
        if row['temperature_c'] is not None:
            packet['outTemp'] = float(row['temperature_c'])
        if row['humidity'] is not None:
            packet['outHumidity'] = float(row['humidity'])
        if row['rainfall_mm'] is not None:
            packet['rain'] = float(row['rainfall_mm']) / 10.0
        if row['wind_kmh'] is not None:
            packet['windSpeed'] = float(row['wind_kmh'])
        if row['wind_direction'] is not None:
            packet['windDir'] = float(row['wind_direction'])

        if packet.get('windDir') is None:
            packet.pop('WindSpeed', None)
        if packet.get('windSpeed') is None or packet['windSpeed'] == 0.0:
            # If no wind, remove direction
            packet.pop('windDir', None)

    def _row_to_packet(self, row, prev_packet=None):
        packet = {}
        if prev_packet is not None:
            for k, v in prev_packet.items():
                packet[k] = v

        # Can't carry this one over.
        packet['rain'] = None

        timestamp = self._dt_to_epoch(row['timestamp'])
        if packet.get('dateTime') == timestamp:
            # Fudge this because weewx uses dateTime as primary key
            packet['dateTime'] += 1
        else:
            packet['dateTime'] = timestamp
        packet['usUnits'] = weewx.METRIC

        if row['sensor'] == self.in_sensor:
            self._update_packet_for_indoor(row, packet)

        if row['sensor'] in self.extra_sensors:
            self._update_packet_for_extra(row, packet)

        if row['sensor'] == 'bridge':
            self._update_packet_for_bridge(row, packet)

        if row['sensor'] == self.out_sensor:
            self._update_packet_for_outdoor(row, packet)

        return packet

    @property
    def hardware_name(self):
        return self._hardware_name

    @property
    def archive_interval(self):
        return self._archive_interval

    def genLoopPackets(self):
        while True:
            rows = self._get_rows_after_row(self._last_row)
            if len(rows) == 0:
                time.sleep(1)
                continue
            for row in rows:
                packet = self._row_to_packet(row, prev_packet=self._last_packet)
                self._last_row = row
                self._last_packet = packet
                yield packet

    def genStartupRecords(self, last_ts):
        if last_ts is None:
            last_ts = 0
            dt = datetime.datetime.utcfromtimestamp(last_ts)
        else:
            dt = datetime.datetime.utcfromtimestamp(last_ts - 300)
        rows = self._get_rows_after_dt(dt)
        if len(rows) == 0:
            return
        for row in rows:
            packet = self._row_to_packet(row, prev_packet=self._last_packet)
            self._last_row = row
            self._last_packet = packet
            if packet['dateTime'] > last_ts:
                yield packet
        # When starting weewx for the first time with a lot of data, it can
        # take quite a while to process everything. More data has come in
        # while we've processed the above records. Let's continue looping
        # until we have everything
        while True:
            rows = self._get_rows_after_row(self._last_row)
            if len(rows) == 0:
                return
            for row in rows:
                packet = self._row_to_packet(row, prev_packet=self._last_packet)
                self._last_row = row
                self._last_packet = packet
                yield packet

    def genArchiveRecords(self, since_ts):
        if since_ts is None:
            since_ts = 0
            dt = datetime.datetime.utcfromtimestamp(since_ts)
        else:
            dt = datetime.datetime.utcfromtimestamp(since_ts - 300)

        prev_packet = None
        rows = self._get_rows_after_dt(dt)
        for row in rows:
            packet = self._row_to_packet(row, prev_packet=prev_packet)
            prev_packet = packet
            if packet['dateTime'] > since_ts:
                yield packet

if __name__ == '__main__':
    db = MySQLdb.connect(
        user='aculink',
        passwd='aculink',
        db='aculink',
        cursorclass=MySQLdb.cursors.DictCursor
    )
    config = {
        'out_sensor': '00001',
        'in_sensor': '00002',
        'extra_sensor1': '00003',
    }

    aculink = Aculink(db=db, config=config)
    for record in aculink.genArchiveRecords(time.time() - 600):
        print record

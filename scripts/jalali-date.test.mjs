import assert from 'node:assert/strict';
import { test } from 'node:test';
import { gregorianDateOnlyToJalali, jalaliDateToGregorianDateOnly, jalaliDateError, normalizeJalaliDate } from '../src/components/date/jalaliDate.js';

test('Jalali calendar keys round-trip without timezone conversion', () => {
  for (const [iso, jalali] of [['2026-11-02', '1405/08/11'], ['2025-03-21', '1404/01/01'], ['2025-03-20', '1403/12/30']]) {
    assert.equal(gregorianDateOnlyToJalali(iso), jalali);
    assert.equal(jalaliDateToGregorianDateOnly(jalali), iso);
  }
  assert.equal(jalaliDateToGregorianDateOnly('۱۴۰۵/۰۸/۱۱'), '2026-11-02');
  assert.equal(jalaliDateToGregorianDateOnly('١٤٠٥/٠٨/١١'), '2026-11-02');
  assert.equal(normalizeJalaliDate('۱۴۰۵۰۸۱۱'), '1405/08/11');
});

test('incomplete, invalid, non-leap and future dates stay invalid', () => {
  for (const date of ['1404/12/30', '1405/13/01', '1405/07/31', '1405//', '9999/01/01']) assert.equal(jalaliDateToGregorianDateOnly(date), '');
  assert.equal(gregorianDateOnlyToJalali('2026-02-30'), '');
  assert.ok(jalaliDateError('1405/08/12', true, undefined, '1405/08/11'));
  assert.ok(jalaliDateError('1405//', true));
  assert.equal(jalaliDateError('1405/08/11', true, undefined, '1405/08/11'), undefined);
});

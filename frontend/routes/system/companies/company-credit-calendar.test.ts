import { describe, expect, test } from 'bun:test';
import { buildCompanyCreditCalendar } from './company-credit-calendar';

const unixDay = (year: number, month: number, day: number): number => (
  Math.floor(Date.UTC(year, month - 1, day) / 86400000)
);

describe('company credit calendar', () => {
  test('builds Monday-first weeks and labels each row from its Monday month', () => {
    const days = Array.from({ length: 30 }, (_, dayOffset) => ({
      Day: unixDay(2026, 8, 16) + dayOffset,
      CPU: 0,
      Inference: 0,
    }));
    const weeks = buildCompanyCreditCalendar(days);

    expect(weeks.map((week) => week.month)).toEqual(['AGO', 'AGO', 'AGO', 'AGO', 'SEP', 'SEP']);
    expect(weeks[0].mondayUnixDay).toBe(unixDay(2026, 8, 10));
    expect(weeks[0].days[6].dayOfMonth).toBe(16);
    expect(weeks[0].days.slice(0, 6).every((cell) => cell.usage === null)).toBe(true);
  });

  test('scales CPU and inference independently across the visible days', () => {
    const weeks = buildCompanyCreditCalendar([
      { Day: unixDay(2026, 8, 17), CPU: 50, Inference: 10 },
      { Day: unixDay(2026, 8, 18), CPU: 100, Inference: 40 },
    ]);
    const monday = weeks[0].days[0];
    const tuesday = weeks[0].days[1];

    expect(monday.cpuPercent).toBe(50);
    expect(monday.inferencePercent).toBe(25);
    expect(tuesday.cpuPercent).toBe(100);
    expect(tuesday.inferencePercent).toBe(100);
  });
});

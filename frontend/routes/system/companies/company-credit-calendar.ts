import type { ICreditUsageDay } from './company-credit-usage.model';

export const COMPANY_CREDIT_WEEKDAY_LABELS = ['L', 'M', 'X', 'J', 'V', 'S', 'D'];

const MONTH_LABELS = ['ENE', 'FEB', 'MAR', 'ABR', 'MAY', 'JUN', 'JUL', 'AGO', 'SEP', 'OCT', 'NOV', 'DIC'];
const MILLISECONDS_PER_DAY = 86400000;

export interface ICompanyCreditCalendarCell {
  unixDay: number;
  dayOfMonth: number;
  usage: ICreditUsageDay | null;
  cpuPercent: number;
  inferencePercent: number;
}

export interface ICompanyCreditCalendarWeek {
  mondayUnixDay: number;
  month: string;
  days: ICompanyCreditCalendarCell[];
}

const mondayOffset = (unixDay: number): number => ((unixDay + 3) % 7 + 7) % 7;
const dateFromUnixDay = (unixDay: number): Date => new Date(unixDay * MILLISECONDS_PER_DAY);
const usagePercent = (value: number, maximum: number): number => (
  maximum > 0 ? Math.max(0, Math.min(100, (value / maximum) * 100)) : 0
);

export const buildCompanyCreditCalendar = (
  sourceDays: ICreditUsageDay[],
): ICompanyCreditCalendarWeek[] => {
  if (!sourceDays.length) return [];

  const days = [...sourceDays].sort((firstDay, secondDay) => firstDay.Day - secondDay.Day);
  const usageByUnixDay = new Map(days.map((day) => [day.Day, day]));
  const firstUnixDay = days[0].Day;
  const lastUnixDay = days[days.length - 1].Day;
  const firstMondayUnixDay = firstUnixDay - mondayOffset(firstUnixDay);
  const lastSundayUnixDay = lastUnixDay + (6 - mondayOffset(lastUnixDay));
  const maximumCPU = days.reduce((maximum, day) => Math.max(maximum, day.CPU || 0), 0);
  const maximumInference = days.reduce((maximum, day) => Math.max(maximum, day.Inference || 0), 0);
  const weeks: ICompanyCreditCalendarWeek[] = [];

  for (let mondayUnixDay = firstMondayUnixDay; mondayUnixDay <= lastSundayUnixDay; mondayUnixDay += 7) {
    const mondayDate = dateFromUnixDay(mondayUnixDay);
    weeks.push({
      mondayUnixDay,
      // A cross-month week belongs to the month containing its Monday.
      month: MONTH_LABELS[mondayDate.getUTCMonth()],
      days: Array.from({ length: 7 }, (_, weekdayIndex) => {
        const unixDay = mondayUnixDay + weekdayIndex;
        const usage = usageByUnixDay.get(unixDay) || null;
        return {
          unixDay,
          dayOfMonth: dateFromUnixDay(unixDay).getUTCDate(),
          usage,
          cpuPercent: usagePercent(usage?.CPU || 0, maximumCPU),
          inferencePercent: usagePercent(usage?.Inference || 0, maximumInference),
        };
      }),
    });
  }

  return weeks;
};

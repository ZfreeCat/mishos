import { kyWrapper } from '@/lib/ky-wrapper';

export const getCalendarToken = async (): Promise<string> => {
  const response: any = await kyWrapper.get('calendar/token').json();
  return response?.token ?? '';
};

export const downloadShowCalendar = async (showId: number | string, fileName: string): Promise<void> => {
  const response: any = await kyWrapper.get(`calendar/show/${showId}.ics`);
  const text = typeof response === 'string' ? response : await response.text();

  const blob = new Blob([text], { type: 'text/calendar;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = fileName;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
};

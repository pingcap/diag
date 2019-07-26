import moment from 'moment';

// oriDatetime: '2019-07-15T06:27:15Z'
export function formatDatetime(oriDatetime: string): string {
  return moment(oriDatetime).format('YYYY-MM-DD hh:mm');
}

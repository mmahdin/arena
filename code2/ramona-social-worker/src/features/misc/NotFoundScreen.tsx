import { useNavigate } from 'react-router-dom';
import { Compass } from 'lucide-react';
import { Screen } from '@/components/layout/Screen';
import { Button } from '@/components/ui/Button';
import { EmptyState } from '@/components/ui/EmptyState';

export function NotFoundScreen() {
  const navigate = useNavigate();
  return (
    <Screen>
      <EmptyState
        icon={<Compass className="size-6" />}
        title="این صفحه وجود ندارد"
        description="ممکن است پیوند تغییر کرده باشد یا دسترسی شما به آن محدود شده باشد."
        action={
          <Button onClick={() => navigate('/')} iconStart={<Compass className="size-4" />}>
            بازگشت به صفحه امروز
          </Button>
        }
      />
    </Screen>
  );
}

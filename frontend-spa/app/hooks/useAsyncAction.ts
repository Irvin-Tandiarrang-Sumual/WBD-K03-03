import { useState } from 'react';

export interface AsyncActionOptions<T> {
  onSuccess?: (result: T) => void | Promise<void>;
  onError?: (error: unknown) => void;
  successMessage?: string | ((result: T) => string);
  errorMessage?: string | ((error: unknown) => string);
  successTone?: 'pine' | 'maroon' | 'blue';
}

export function useAsyncAction(
  showSnackbar?: (message: string, tone?: 'pine' | 'maroon' | 'blue') => void
) {
  const [isLoading, setIsLoading] = useState(false);

  const execute = async <T>(
    actionFn: () => Promise<T>,
    options?: AsyncActionOptions<T>
  ): Promise<T | null> => {
    setIsLoading(true);
    try {
      const result = await actionFn();
      if (options?.successMessage && showSnackbar) {
        const msg =
          typeof options.successMessage === 'function'
            ? options.successMessage(result)
            : options.successMessage;
        showSnackbar(msg, options.successTone ?? 'pine');
      }
      if (options?.onSuccess) {
        await options.onSuccess(result);
      }
      return result;
    } catch (err) {
      if (showSnackbar) {
        const errorMsg = options?.errorMessage
          ? typeof options.errorMessage === 'function'
            ? options.errorMessage(err)
            : options.errorMessage
          : err instanceof Error
          ? err.message
          : 'An unexpected error occurred';
        showSnackbar(errorMsg, 'maroon');
      }
      options?.onError?.(err);
      return null;
    } finally {
      setIsLoading(false);
    }
  };

  return { isLoading, execute };
}

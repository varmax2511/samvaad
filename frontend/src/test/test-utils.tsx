import { ReactNode } from 'react';
import { render, RenderOptions } from '@testing-library/react';
import { BrowserRouter, MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '../context/AuthContext';

interface WrapperProps {
  children: ReactNode;
}

function AllProviders({ children }: WrapperProps) {
  return (
    <BrowserRouter>
      <AuthProvider>{children}</AuthProvider>
    </BrowserRouter>
  );
}

interface MemoryRouterWrapperOptions {
  initialEntries?: string[];
}

function createMemoryRouterWrapper({ initialEntries = ['/'] }: MemoryRouterWrapperOptions = {}) {
  return function MemoryRouterWrapper({ children }: WrapperProps) {
    return (
      <MemoryRouter initialEntries={initialEntries}>
        <AuthProvider>{children}</AuthProvider>
      </MemoryRouter>
    );
  };
}

function customRender(ui: React.ReactElement, options?: Omit<RenderOptions, 'wrapper'>) {
  return render(ui, { wrapper: AllProviders, ...options });
}

function renderWithMemoryRouter(
  ui: React.ReactElement,
  { initialEntries = ['/'], ...options }: MemoryRouterWrapperOptions & Omit<RenderOptions, 'wrapper'> = {}
) {
  return render(ui, {
    wrapper: createMemoryRouterWrapper({ initialEntries }),
    ...options,
  });
}

export * from '@testing-library/react';
export { customRender as render, renderWithMemoryRouter };

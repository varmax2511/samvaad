import { describe, it, expect, beforeEach } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithMemoryRouter } from '../test/test-utils';
import ProtectedRoute from './ProtectedRoute';

describe('ProtectedRoute', () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it('should redirect to login when not authenticated', () => {
    renderWithMemoryRouter(
      <ProtectedRoute>
        <div>Protected Content</div>
      </ProtectedRoute>,
      { initialEntries: ['/home'] }
    );

    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
  });

  it('should render children when authenticated', () => {
    sessionStorage.setItem('samvaad_user', JSON.stringify({ id: 'test-id', name: 'Test User' }));

    renderWithMemoryRouter(
      <ProtectedRoute>
        <div>Protected Content</div>
      </ProtectedRoute>,
      { initialEntries: ['/home'] }
    );

    expect(screen.getByText('Protected Content')).toBeInTheDocument();
  });
});

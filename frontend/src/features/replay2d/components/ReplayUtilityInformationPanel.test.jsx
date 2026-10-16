import {render, screen} from '@testing-library/react';
import {describe, it, expect} from 'vitest';
import {ReplayUtilityInformationPanel} from './ReplayUtilityInformationPanel';

describe('visual utility limits',()=>{
  it('presents the five-second deduction with unknown side/origin and excluded flashes',()=>{
    render(<ReplayUtilityInformationPanel tickRate={64} information={{source_status:'estimated_source_supplied',
      observations:[],objects:[{object_id:'1',utility_type:'he',kind:'effect',age_ticks:64,state:'last_observed'}],by_player:[],
      launcher_estimates:[{id:'group',utility_type:'he',first_launch_tick:100,last_launch_tick:420}]}} />);
    expect(screen.getByText(/indicio de al menos tres lanzadores/)).toHaveTextContent('5 s');
    expect(screen.getByText(/Bando, origen/)).toHaveTextContent('flashes se excluyen');
    expect(screen.getByText(/Posición actual desconocida/)).toBeInTheDocument();
  });
  it('shows unavailable source without asserting no grenades',()=>{
    render(<ReplayUtilityInformationPanel information={{source_status:'unavailable',objects:[],observations:[],launcher_estimates:[],by_player:[]}} />);
    expect(screen.getByText(/Fuente visual de utilidad no disponible/)).toBeInTheDocument();
    expect(screen.queryByText(/No hay señales visuales registradas/)).not.toBeInTheDocument();
  });
});

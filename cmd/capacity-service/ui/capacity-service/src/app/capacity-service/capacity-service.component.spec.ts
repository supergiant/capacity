import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { CapacityServiceComponent } from './capacity-service.component';

describe('CapacityServiceComponent', () => {
  let component: CapacityServiceComponent;
  let fixture: ComponentFixture<CapacityServiceComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ CapacityServiceComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(CapacityServiceComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

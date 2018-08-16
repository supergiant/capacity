import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { NodeTypesModalComponent } from './node-types-modal.component';

describe('NodeTypesModalComponent', () => {
  let component: NodeTypesModalComponent;
  let fixture: ComponentFixture<NodeTypesModalComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ NodeTypesModalComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(NodeTypesModalComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
